// In build/init.c
#define _GNU_SOURCE
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <fcntl.h>
#include <sys/wait.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/reboot.h>
#include <sys/sysmacros.h>
#include <limits.h>

// A proper shutdown function
__attribute__((noreturn)) static void poweroff(void) {
    fflush(stdout);
    fflush(stderr);
    reboot(RB_POWER_OFF);
    // If reboot fails, we're in big trouble.
    // Loop forever to prevent the kernel from panicking on our exit.
    for (;;) {
        sleep(3600);
    }
}

// A proper panic function for our init
static void panic(const char *what) {
    fprintf(stderr, "\n\nINIT PANIC: %s: %s\n\n", what, strerror(errno));
    poweroff();
}

// Ensure the console is available for logging
static void ensure_console(void) {
    // Create /dev/console if it doesn't exist (it should)
    if (mknod("/dev/console", S_IFCHR | 0600, makedev(5, 1)) && errno != EEXIST)
        panic("mknod(/dev/console)");

    int fd = open("/dev/console", O_RDWR);
    if (fd < 0)
        panic("open(/dev/console)");

    // Redirect stdin, stdout, and stderr to the console
    dup2(fd, 0);
    dup2(fd, 1);
    dup2(fd, 2);
    if (fd > 2)
        close(fd);
}

// A more robust filesystem setup
static void mount_filesystems(void) {
    // Mount the essentials for the Go runtime and other tools
    if (mount("proc", "/proc", "proc", 0, NULL))
        panic("mount(/proc)");
    if (mount("sysfs", "/sys", "sysfs", 0, NULL))
        panic("mount(/sys)");
    if (mount("devtmpfs", "/dev", "devtmpfs", 0, NULL))
        panic("mount(/dev)");

    // Create and mount tmpfs for runtime data
    mkdir("/tmp", 0777);
    if (mount("tmpfs", "/tmp", "tmpfs", 0, NULL))
        panic("mount(/tmp)");
    mkdir("/run", 0755);
    if (mount("tmpfs", "/run", "tmpfs", 0, NULL))
        panic("mount(/run)");
}

// Load a kernel module by name (without .ko extension)
// Returns 0 on success, -1 on failure
static int load_module(const char *module_name) {
    char module_path[PATH_MAX];
    int fd;
    struct stat st;
    void *module_data;
    int ret;

    // Try different module paths and extensions
    const char *module_paths[] = {
        "/lib/modules/%s.ko",
        "/lib/modules/%s.ko.gz",
        "/lib/modules/%s.ko.xz",
        "/lib/modules/%s.ko.zst",
        NULL
    };

    for (int i = 0; module_paths[i]; i++) {
        snprintf(module_path, sizeof(module_path), module_paths[i], module_name);

        if (stat(module_path, &st) != 0)
            continue;

        printf("C INIT: Found module %s at %s\n", module_name, module_path);

        fd = open(module_path, O_RDONLY);
        if (fd < 0) {
            fprintf(stderr, "C INIT: Failed to open %s: %s\n", module_path, strerror(errno));
            continue;
        }

        module_data = malloc(st.st_size);
        if (!module_data) {
            fprintf(stderr, "C INIT: Failed to allocate memory for %s\n", module_name);
            close(fd);
            continue;
        }

        if (read(fd, module_data, st.st_size) != st.st_size) {
            fprintf(stderr, "C INIT: Failed to read %s\n", module_path);
            free(module_data);
            close(fd);
            continue;
        }
        close(fd);

        // Try to load the module using init_module syscall (175 on x86_64)
        ret = syscall(175, module_data, st.st_size, "");
        if (ret != 0 && errno != EEXIST) {
            fprintf(stderr, "C INIT: init_module(%s) failed: %s\n", module_name, strerror(errno));
        }
        free(module_data);

        if (ret == 0 || errno == EEXIST) {
            printf("C INIT: Loaded kernel module %s\n", module_name);
            return 0;
        }
    }

    fprintf(stderr, "C INIT: Module %s not found in initramfs\n", module_name);
    return -1;
}

// Load required filesystem modules for squashfs and overlayfs
static void load_filesystem_modules(void) {
    printf("C INIT: Loading filesystem modules...\n");

    // Load squashfs module
    if (load_module("squashfs") != 0) {
        printf("C INIT: squashfs module not loaded (may be built-in)\n");
    }

    // Load overlay module
    if (load_module("overlay") != 0) {
        printf("C INIT: overlay module not loaded (may be built-in)\n");
    }
}

// Helper to read a kernel cmdline parameter into caller-provided buffer.
// Returns 1 when present, 0 otherwise.
static int get_cmdline_param(const char *key, char *value, size_t value_len) {
    if (!value || value_len == 0) {
        return 0;
    }
    value[0] = '\0';

    FILE *f = fopen("/proc/cmdline", "r");
    if (!f)
        return 0;
    
    char line[4096];
    if (!fgets(line, sizeof(line), f)) {
        fclose(f);
        return 0;
    }
    fclose(f);
    
    // Search for key=value
    char search[256];
    snprintf(search, sizeof(search), "%s=", key);
    char *found = strstr(line, search);
    if (!found)
        return 0;
    
    found += strlen(search);
    char *end = strchr(found, ' ');
    size_t len = end ? (size_t)(end - found) : strlen(found);
    if (len >= value_len) len = value_len - 1;
    
    strncpy(value, found, len);
    value[len] = '\0';
    
    // Trim newline
    char *nl = strchr(value, '\n');
    if (nl) *nl = '\0';
    
    return 1;
}

// Try to mount rootfs based on standard Linux kernel parameters (root= and rootfstype=)
// Returns 1 if mounted successfully, 0 if no rootfs specified or failed
static int try_mount_rootfs(void) {
    // Read standard Linux kernel parameters (not volant-specific!)
    char root_device_raw[256];
    if (!get_cmdline_param("root", root_device_raw, sizeof(root_device_raw)) ||
        strlen(root_device_raw) == 0) {
        printf("C INIT: No root= parameter, staying in initramfs\n");
        return 0;
    }
    
    char rootfs_fstype[64];
    if (!get_cmdline_param("rootfstype", rootfs_fstype, sizeof(rootfs_fstype)) ||
        strlen(rootfs_fstype) == 0) {
        strncpy(rootfs_fstype, "ext4", sizeof(rootfs_fstype) - 1);
        rootfs_fstype[sizeof(rootfs_fstype) - 1] = '\0';
    }
    
    char normalized_device[PATH_MAX];
    const char *root_device = root_device_raw;
    if (strncmp(root_device_raw, "/dev/", 5) != 0 &&
        strchr(root_device_raw, '=') == NULL &&
        strchr(root_device_raw, '/') == NULL) {
        snprintf(normalized_device, sizeof(normalized_device), "/dev/%s", root_device_raw);
        root_device = normalized_device;
    }

    printf("C INIT: Mounting root=%s rootfstype=%s\n", root_device, rootfs_fstype);
    
    // Wait for device (up to 5 seconds) when the root looks like a block device path.
    if (strncmp(root_device, "/dev/", 5) == 0) {
        for (int i = 0; i < 50; i++) {
            struct stat st;
            if (stat(root_device, &st) == 0 && S_ISBLK(st.st_mode)) {
                break;
            }
            usleep(100000); // 100ms
        }
    }
    
    if (strncmp(root_device, "/dev/", 5) == 0) {
        struct stat st;
        if (stat(root_device, &st) != 0 || !S_ISBLK(st.st_mode)) {
            fprintf(stderr, "C INIT: Device %s not found, staying in initramfs\n", root_device);
            return 0;
        }
    }
    
    mkdir("/newroot", 0755);

    if (strcmp(rootfs_fstype, "squashfs") == 0) {
        printf("C INIT: Setting up squashfs with overlayfs\n");

        // Load required kernel modules
        load_filesystem_modules();

        // Mount squashfs as lower layer (read-only)
        mkdir("/lower", 0755);
        if (mount(root_device, "/lower", "squashfs", MS_RDONLY, NULL)) {
            fprintf(stderr, "C INIT: Failed to mount squashfs: %s\n", strerror(errno));
            rmdir("/lower");
            rmdir("/newroot");
            return 0;
        }
        
        // Get overlay size from cmdline (default 1G)
        char overlay_size[64];
        if (!get_cmdline_param("overlay_size", overlay_size, sizeof(overlay_size)) ||
            strlen(overlay_size) == 0) {
            strncpy(overlay_size, "1G", sizeof(overlay_size) - 1);
            overlay_size[sizeof(overlay_size) - 1] = '\0';
        }
        
        // Create upper directory and mount tmpfs to it
        mkdir("/upper", 0755);
        
        char tmpfs_opts[256];
        snprintf(tmpfs_opts, sizeof(tmpfs_opts), "size=%s", overlay_size);
        
        if (mount("tmpfs", "/upper", "tmpfs", 0, tmpfs_opts)) {
            fprintf(stderr, "C INIT: Failed to mount tmpfs upper: %s\n", strerror(errno));
            umount("/lower");
            return 0;
        }
        
        // CRITICAL: Create work directory INSIDE the tmpfs mount
        // overlayfs requires workdir to be on same filesystem as upperdir
        mkdir("/upper/work", 0755);
        
        // Mount overlayfs
        if (mount("overlay", "/newroot", "overlay", 0, 
                  "lowerdir=/lower,upperdir=/upper,workdir=/upper/work")) {
            fprintf(stderr, "C INIT: Failed to mount overlayfs: %s\n", strerror(errno));
            umount("/upper");
            umount("/lower");
            return 0;
        }
        
        printf("C INIT: overlayfs mounted (squashfs lower + tmpfs upper %s)\n", overlay_size);
    } else {
        // Legacy ext4/xfs/btrfs direct mount
        if (mount(root_device, "/newroot", rootfs_fstype, 0, NULL)) {
            fprintf(stderr, "C INIT: Failed to mount %s rootfs: %s\n", rootfs_fstype, strerror(errno));
            rmdir("/newroot");
            return 0;
        }
        printf("C INIT: mounted %s rootfs\n", rootfs_fstype);
    }
    
    // Chroot into newroot
    if (chdir("/newroot")) panic("chdir(/newroot)");
    if (chroot(".")) panic("chroot(.)");
    if (chdir("/")) panic("chdir(/)");
    
    // Re-mount essential filesystems in new root
    if (mount("proc", "/proc", "proc", 0, NULL)) panic("mount(/proc) in newroot");
    if (mount("sysfs", "/sys", "sysfs", 0, NULL)) panic("mount(/sys) in newroot");
    if (mount("devtmpfs", "/dev", "devtmpfs", 0, NULL)) panic("mount(/dev) in newroot");
    mkdir("/tmp", 0777);
    if (mount("tmpfs", "/tmp", "tmpfs", 0, NULL)) panic("mount(/tmp) in newroot");
    mkdir("/run", 0755);
    if (mount("tmpfs", "/run", "tmpfs", 0, NULL)) panic("mount(/run) in newroot");
    
    return 1;
}

int main(int argc, char *argv[]) {
    // Create a basic directory structure first
    mkdir("/proc", 0755);
    mkdir("/sys", 0755);
    mkdir("/dev", 0755);
    mkdir("/bin", 0755);
    mkdir("/sbin", 0755);

    // Set up the essential filesystems
    mount_filesystems();

    // Now that /dev is mounted, ensure we have a console
    ensure_console();

    printf("C INIT: Basic environment is up.\n");
    
    // Try to mount rootfs if kernel cmdline specifies root= and rootfstype=
    // This is standard Linux behavior - if root= exists, mount it
    int mounted_rootfs = try_mount_rootfs();
    if (mounted_rootfs) {
        printf("C INIT: Rootfs mounted and pivoted.\n");
    } else {
        printf("C INIT: No rootfs specified, staying in initramfs.\n");
    }

    // Standard Linux init search order (try multiple possible init locations)
    // This makes C init work with ANY userspace init, not just kestrel
    printf("C INIT: Searching for init...\n");
    
    const char *init_paths[] = {
        "/sbin/init",       // Standard systemd/sysvinit location
        "/init",            // Custom init in rootfs root
        "/bin/init",        // Alternative location
        "/bin/kestrel",     // Volant's Go init
        NULL
    };
    
    for (int i = 0; init_paths[i] != NULL; i++) {
        struct stat st;
        if (stat(init_paths[i], &st) == 0 && (st.st_mode & S_IXUSR)) {
            printf("C INIT: Found init at %s, executing...\n", init_paths[i]);
            char *const init_argv[] = {(char *)init_paths[i], NULL};
            execv(init_paths[i], init_argv);
            // If execv returns, it failed - try next
            fprintf(stderr, "C INIT: Failed to exec %s: %s\n", init_paths[i], strerror(errno));
        }
    }

    // No init found anywhere - this is a catastrophe
    panic("No init found in /sbin/init, /init, /bin/init, or /bin/kestrel");

    return 1; // Unreachable
}

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

// Helper to read kernel cmdline parameter
static const char* get_cmdline_param(const char *key) {
    static char value[256];
    FILE *f = fopen("/proc/cmdline", "r");
    if (!f) return NULL;
    
    char line[4096];
    if (!fgets(line, sizeof(line), f)) {
        fclose(f);
        return NULL;
    }
    fclose(f);
    
    // Search for key=value
    char search[256];
    snprintf(search, sizeof(search), "%s=", key);
    char *found = strstr(line, search);
    if (!found) return NULL;
    
    found += strlen(search);
    char *end = strchr(found, ' ');
    size_t len = end ? (size_t)(end - found) : strlen(found);
    if (len >= sizeof(value)) len = sizeof(value) - 1;
    
    strncpy(value, found, len);
    value[len] = '\0';
    
    // Trim newline
    char *nl = strchr(value, '\n');
    if (nl) *nl = '\0';
    
    return value;
}

// Try to mount rootfs (squashfs with overlayfs, or legacy ext4/xfs/btrfs)
static int try_mount_rootfs(void) {
    const char *rootfs_fstype = get_cmdline_param("rootfs_fstype");
    if (!rootfs_fstype) rootfs_fstype = "ext4"; // default
    
    const char *root_device = "/dev/vda";
    
    printf("C INIT: rootfs_fstype=%s device=%s\n", rootfs_fstype, root_device);
    
    // Wait for device (up to 5 seconds)
    for (int i = 0; i < 50; i++) {
        struct stat st;
        if (stat(root_device, &st) == 0 && S_ISBLK(st.st_mode)) {
            break;
        }
        usleep(100000); // 100ms
    }
    
    struct stat st;
    if (stat(root_device, &st) != 0 || !S_ISBLK(st.st_mode)) {
        printf("C INIT: No rootfs device found, staying in initramfs\n");
        return 0;
    }
    
    mkdir("/newroot", 0755);
    
    if (strcmp(rootfs_fstype, "squashfs") == 0) {
        printf("C INIT: Setting up squashfs with overlayfs\n");
        
        // Mount squashfs as lower layer (read-only)
        mkdir("/lower", 0755);
        if (mount(root_device, "/lower", "squashfs", MS_RDONLY, NULL)) {
            fprintf(stderr, "C INIT: Failed to mount squashfs: %s\n", strerror(errno));
            rmdir("/lower");
            rmdir("/newroot");
            return 0;
        }
        
        // Create upper and work dirs for overlayfs
        mkdir("/upper", 0755);
        mkdir("/work", 0755);
        
        // Get overlay size from cmdline (default 1G)
        const char *overlay_size = get_cmdline_param("overlay_size");
        if (!overlay_size) overlay_size = "1G";
        
        char tmpfs_opts[256];
        snprintf(tmpfs_opts, sizeof(tmpfs_opts), "size=%s", overlay_size);
        
        if (mount("tmpfs", "/upper", "tmpfs", 0, tmpfs_opts)) {
            fprintf(stderr, "C INIT: Failed to mount tmpfs upper: %s\n", strerror(errno));
            umount("/lower");
            return 0;
        }
        
        // Mount overlayfs
        if (mount("overlay", "/newroot", "overlay", 0, 
                  "lowerdir=/lower,upperdir=/upper,workdir=/work")) {
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
    mkdir("/bin", 0755); // For kestrel

    // Set up the essential filesystems
    mount_filesystems();

    // Now that /dev is mounted, ensure we have a console
    ensure_console();

    printf("C INIT: Basic environment is up.\n");
    
    // Try to mount rootfs if available (squashfs, ext4, xfs, btrfs)
    // If successful, we'll chroot into it
    int mounted_rootfs = try_mount_rootfs();
    if (mounted_rootfs) {
        printf("C INIT: Pivoted to rootfs. Handing off to kestrel...\n");
    } else {
        printf("C INIT: Staying in initramfs. Handing off to kestrel...\n");
    }

    // Hand over control to our real Go init program.
    // This will now become the new PID 1.
    char *const kestrel_argv[] = {"/bin/kestrel", NULL};
    execv("/bin/kestrel", kestrel_argv);

    // If execv returns, it failed. This is a catastrophe.
    panic("execv(/bin/kestrel)");

    return 1; // Unreachable
}

// kestrel.c - Unified init + workload supervisor + API server for Volant microVMs
// Copyright (c) 2025 HYPR. PTE. LTD.
// Business Source License 1.1
//
// This single C program handles:
// 1. Early boot (PID 1 init responsibilities)
// 2. Rootfs mounting (squashfs + overlayfs, ext4, etc.)
// 3. Environment setup (DNS, env vars from kernel cmdline)
// 4. Manifest parsing (from cmdline or API fetch)
// 5. Workload execution and supervision
// 6. HTTP API server (health checks, reverse proxy)

#define _GNU_SOURCE
#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <fcntl.h>
#include <signal.h>
#include <time.h>
#include <ctype.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <sys/mount.h>
#include <sys/reboot.h>
#include <sys/socket.h>
#include <sys/select.h>
#ifdef __linux__
#include <sys/sysmacros.h>
#endif
#include <netinet/in.h>
#include <arpa/inet.h>
#include <zlib.h>

// ============================================================================
// CONFIGURATION & CONSTANTS
// ============================================================================

#define KESTREL_VERSION "2.0.0"
#define API_PORT 8080
#define MAX_CMDLINE_PARAMS 256
#define MAX_ENV_VARS 128
#define MAX_WORKLOAD_RESTARTS 10
#define WORKLOAD_RESTART_DELAY_SEC 5
#define HEALTH_CHECK_INTERVAL_MS 50
#define HEALTH_CHECK_TIMEOUT_SEC 120

// ============================================================================
// DATA STRUCTURES
// ============================================================================

// Kernel cmdline parameter storage
typedef struct {
    char *key;
    char *value;
} cmdline_param_t;

typedef struct {
    cmdline_param_t params[MAX_CMDLINE_PARAMS];
    int count;
} cmdline_t;

// Manifest structures
typedef struct {
    char **entrypoint;     // NULL-terminated array
    int entrypoint_count;
    char **env;            // NULL-terminated array of "KEY=VALUE"
    int env_count;
    char *type;            // "exec", "http", "grpc"
    char *base_url;        // For HTTP workloads
    char *workdir;
    char *health_endpoint; // Health check path
    int health_timeout_ms;
} workload_spec_t;

typedef struct {
    char *name;
    char *version;
    char *runtime;
    workload_spec_t workload;
} manifest_t;

// Global state
typedef struct {
    cmdline_t cmdline;
    manifest_t *manifest;
    pid_t workload_pid;
    int workload_restart_count;
    time_t start_time;
    int api_socket;
    int should_exit;
} kestrel_state_t;

static kestrel_state_t g_state = {0};

// ============================================================================
// LOGGING & UTILITIES
// ============================================================================

#define LOG(fmt, ...) do { \
    time_t now = time(NULL); \
    fprintf(stderr, "[kestrel %.24s] " fmt "\n", ctime(&now), ##__VA_ARGS__); \
    fflush(stderr); \
} while(0)

#define LOG_BOOT(fmt, ...) do { \
    fprintf(stderr, "[kestrel-boot] " fmt "\n", ##__VA_ARGS__); \
    fflush(stderr); \
} while(0)

#define FATAL(fmt, ...) do { \
    LOG("FATAL: " fmt, ##__VA_ARGS__); \
    if (getpid() == 1) { \
        LOG("Cannot exit as PID 1, halting"); \
        for(;;) sleep(3600); \
    } \
    exit(1); \
} while(0)

__attribute__((noreturn)) static void panic(const char *msg) {
    fprintf(stderr, "\n\nKESTREL PANIC: %s: %s\n\n", msg, strerror(errno));
    fflush(stderr);
    reboot(RB_POWER_OFF);
    for(;;) sleep(3600);
}

// Removed: trim_whitespace - not needed

static char* xstrdup(const char *s) {
    if (!s) return NULL;
    char *dup = strdup(s);
    if (!dup) FATAL("strdup failed: out of memory");
    return dup;
}

// ============================================================================
// EARLY BOOT: FILESYSTEM SETUP
// ============================================================================

static void ensure_console(void) {
    if (mknod("/dev/console", S_IFCHR | 0600, makedev(5, 1)) && errno != EEXIST) {
        panic("mknod(/dev/console)");
    }

    int fd = open("/dev/console", O_RDWR);
    if (fd < 0) panic("open(/dev/console)");

    dup2(fd, 0);
    dup2(fd, 1);
    dup2(fd, 2);
    if (fd > 2) close(fd);
}

static void mount_essential_filesystems(void) {
    mkdir("/proc", 0755);
    mkdir("/sys", 0755);
    mkdir("/dev", 0755);
    mkdir("/tmp", 0777);
    mkdir("/run", 0755);

    if (mount("proc", "/proc", "proc", 0, NULL) && errno != EBUSY) {
        panic("mount(/proc)");
    }
    if (mount("sysfs", "/sys", "sysfs", 0, NULL) && errno != EBUSY) {
        panic("mount(/sys)");
    }
    if (mount("devtmpfs", "/dev", "devtmpfs", 0, NULL) && errno != EBUSY) {
        panic("mount(/dev)");
    }
    if (mount("tmpfs", "/tmp", "tmpfs", 0, NULL) && errno != EBUSY) {
        panic("mount(/tmp)");
    }
    if (mount("tmpfs", "/run", "tmpfs", 0, NULL) && errno != EBUSY) {
        panic("mount(/run)");
    }

    // Create /dev/fd and standard stream symlinks for process substitution
    if (symlink("/proc/self/fd", "/dev/fd") && errno != EEXIST) {
        // Non-fatal, but log if verbose
    }
    if (symlink("/proc/self/fd/0", "/dev/stdin") && errno != EEXIST) {
        // Non-fatal
    }
    if (symlink("/proc/self/fd/1", "/dev/stdout") && errno != EEXIST) {
        // Non-fatal
    }
    if (symlink("/proc/self/fd/2", "/dev/stderr") && errno != EEXIST) {
        // Non-fatal
    }
}

// ============================================================================
// KERNEL MODULE LOADING
// ============================================================================

static int load_kernel_module(const char *module_name) {
    char module_path[256];
    const char *paths[] = {
        "/lib/modules/%s.ko",
        "/lib/modules/%s.ko.gz",
        "/lib/modules/%s.ko.xz",
        "/lib/modules/%s.ko.zst",
        NULL
    };

    for (int i = 0; paths[i]; i++) {
        snprintf(module_path, sizeof(module_path), paths[i], module_name);

        struct stat st;
        if (stat(module_path, &st) != 0) continue;

        int fd = open(module_path, O_RDONLY);
        if (fd < 0) continue;

        void *data = malloc(st.st_size);
        if (!data) {
            close(fd);
            continue;
        }

        if (read(fd, data, st.st_size) != st.st_size) {
            free(data);
            close(fd);
            continue;
        }
        close(fd);

        // init_module syscall (175 on x86_64)
        int ret = syscall(175, data, st.st_size, "");
        free(data);

        if (ret == 0 || errno == EEXIST) {
            LOG_BOOT("Loaded kernel module: %s", module_name);
            return 0;
        }
    }

    LOG_BOOT("Module %s not found or failed to load (may be built-in)", module_name);
    return -1;
}

static void load_filesystem_modules(void) {
    load_kernel_module("squashfs");
    load_kernel_module("overlay");
}

// ============================================================================
// KERNEL CMDLINE PARSING
// ============================================================================

static void parse_cmdline(cmdline_t *cmd) {
    cmd->count = 0;

    FILE *f = fopen("/proc/cmdline", "r");
    if (!f) {
        LOG_BOOT("Warning: cannot read /proc/cmdline");
        return;
    }

    char line[4096];
    if (!fgets(line, sizeof(line), f)) {
        fclose(f);
        return;
    }
    fclose(f);

    char *saveptr = NULL;
    char *token = strtok_r(line, " \t\n", &saveptr);

    while (token && cmd->count < MAX_CMDLINE_PARAMS) {
        char *eq = strchr(token, '=');
        if (eq) {
            *eq = '\0';
            cmd->params[cmd->count].key = xstrdup(token);
            cmd->params[cmd->count].value = xstrdup(eq + 1);
        } else {
            cmd->params[cmd->count].key = xstrdup(token);
            cmd->params[cmd->count].value = NULL;
        }
        cmd->count++;
        token = strtok_r(NULL, " \t\n", &saveptr);
    }

    LOG_BOOT("Parsed %d kernel cmdline parameters", cmd->count);
    if (cmd->count > 0) LOG_BOOT("param0: %s=%s", cmd->params[0].key, cmd->params[0].value);
    if (cmd->count > 1) LOG_BOOT("param1: %s=%s", cmd->params[1].key, cmd->params[1].value);
    if (cmd->count > 2) LOG_BOOT("param2: %s=%s", cmd->params[2].key, cmd->params[2].value);
    if (cmd->count > 3) LOG_BOOT("param3: %s=%s", cmd->params[3].key, cmd->params[3].value);
}

static const char* cmdline_get(const cmdline_t *cmd, const char *key) {
    for (int i = 0; i < cmd->count; i++) {
        if (strcmp(cmd->params[i].key, key) == 0) {
            return cmd->params[i].value;
        }
    }
    return NULL;
}

static const char* cmdline_get_or_default(const cmdline_t *cmd, const char *key, const char *def) {
    const char *val = cmdline_get(cmd, key);
    return val ? val : def;
}

// ============================================================================
// BASE64URL DECODING (for volant.env and volant.manifest)
// Volant uses base64url (RFC 4648) without padding
// ============================================================================

static const char base64url_chars[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";

static size_t base64_decode(const char *input, unsigned char **output) {
    size_t input_len = strlen(input);

    // Add padding if needed for base64url
    size_t padding = (4 - (input_len % 4)) % 4;

    // Calculate output length
    size_t output_len = ((input_len + padding) / 4) * 3;
    if (padding > 0) output_len -= padding;

    *output = malloc(output_len + 1);
    if (!*output) return 0;

    int vals[4];
    size_t out_idx = 0;
    size_t i = 0;

    while (i < input_len) {
        // Read 4 characters (or less for last group)
        for (int j = 0; j < 4; j++) {
            if (i + j < input_len) {
                const char *ptr = strchr(base64url_chars, input[i + j]);
                vals[j] = ptr ? (ptr - base64url_chars) : 0;
            } else {
                vals[j] = 0;
            }
        }

        // Decode to 3 bytes
        (*output)[out_idx++] = (vals[0] << 2) | (vals[1] >> 4);
        if (out_idx < output_len) (*output)[out_idx++] = (vals[1] << 4) | (vals[2] >> 2);
        if (out_idx < output_len) (*output)[out_idx++] = (vals[2] << 6) | vals[3];

        i += 4;
    }

    (*output)[out_idx] = '\0';
    return out_idx;
}

// ============================================================================
// ENVIRONMENT SETUP (from kernel cmdline)
// ============================================================================

static void setup_environment_from_cmdline(const cmdline_t *cmd) {
    // Parse volant.env (base64-encoded JSON: {"KEY":"VALUE",...})
    const char *env_encoded = cmdline_get(cmd, "volant.env");
    if (!env_encoded) {
        LOG_BOOT("No volant.env found in cmdline");
        return;
    }

    LOG_BOOT("Found volant.env: %s", env_encoded);

    unsigned char *env_json = NULL;
    size_t json_len = base64_decode(env_encoded, &env_json);

    if (json_len == 0 || !env_json) {
        LOG_BOOT("Warning: failed to decode volant.env");
        return;
    }

    LOG_BOOT("Decoded env JSON (%zu bytes): %s", json_len, (char*)env_json);

    // Simple JSON parser for {"key":"value", ...}
    // This is minimal - for production, use cJSON or similar
    int env_count = 0;
    char *json_str = (char*)env_json;
    char *start = strchr(json_str, '{');
    if (!start) {
        free(env_json);
        return;
    }

    // Parse key-value pairs manually (simplified JSON parser)
    char *ptr = start + 1;
    while (*ptr && *ptr != '}') {
        // Skip whitespace, quotes, and commas
        while (*ptr && (isspace(*ptr) || *ptr == '"' || *ptr == ',')) ptr++;
        if (!*ptr || *ptr == '}') break;

        // Extract key (between quotes)
        char *key_start = ptr;
        while (*ptr && *ptr != '"' && *ptr != ':') ptr++;
        size_t key_len = ptr - key_start;

        // Skip to colon
        while (*ptr && *ptr != ':') ptr++;
        if (!*ptr) break;
        ptr++; // Skip ':'

        // Skip whitespace and opening quote of value
        while (*ptr && (isspace(*ptr) || *ptr == '"')) ptr++;

        // Extract value (until closing quote)
        char *val_start = ptr;
        while (*ptr && *ptr != '"') ptr++;
        size_t val_len = ptr - val_start;

        // Set environment variable (copy strings to not corrupt JSON)
        if (key_len > 0 && val_len > 0) {
            char *key = strndup(key_start, key_len);
            char *val = strndup(val_start, val_len);
            if (key && val) {
                setenv(key, val, 1);
                LOG_BOOT("Set env: %s=%s", key, val);
                env_count++;
                free(key);
                free(val);
            }
        }

        // Move past closing quote
        if (*ptr == '"') ptr++;
    }

    free(env_json);
    LOG_BOOT("Set %d environment variables from kernel cmdline", env_count);
}

static void setup_dns_from_cmdline(const cmdline_t *cmd) {
    const char *dns_server = cmdline_get(cmd, "volant.dns_server");
    const char *dns_search = cmdline_get(cmd, "volant.dns_search");

    if (!dns_server) return;

    mkdir("/etc", 0755);

    FILE *f = fopen("/etc/resolv.conf", "w");
    if (!f) {
        LOG_BOOT("Warning: failed to write /etc/resolv.conf");
        return;
    }

    fprintf(f, "nameserver %s\n", dns_server);
    fprintf(f, "nameserver 1.1.1.1\n");
    fprintf(f, "nameserver 8.8.8.8\n");
    if (dns_search) {
        fprintf(f, "search %s\n", dns_search);
    }
    fclose(f);

    LOG_BOOT("Configured DNS: nameserver=%s search=%s fallback=1.1.1.1,8.8.8.8", dns_server, dns_search ? dns_search : "(none)");
}

// ============================================================================
// ROOTFS MOUNTING
// ============================================================================

static int wait_for_device(const char *device, int timeout_sec) {
    LOG_BOOT("Waiting for device: %s", device);

    for (int i = 0; i < timeout_sec * 10; i++) {
        struct stat st;
        if (stat(device, &st) == 0 && S_ISBLK(st.st_mode)) {
            LOG_BOOT("Device %s is ready", device);
            return 0;
        }
        usleep(100000); // 100ms
    }

    LOG_BOOT("Device %s not found after %d seconds", device, timeout_sec);
    return -1;
}

static int mount_rootfs_squashfs(const char *device, const cmdline_t *cmd) {
    LOG_BOOT("Setting up squashfs with overlayfs");

    load_filesystem_modules();

    // Mount points
    mkdir("/lower", 0755);
    mkdir("/upper", 0755);
    mkdir("/newroot", 0755);

    // Mount squashfs as lower (read-only)
    if (mount(device, "/lower", "squashfs", MS_RDONLY, NULL)) {
        LOG_BOOT("Failed to mount squashfs: %s", strerror(errno));
        return -1;
    }

    // Get overlay size from cmdline
    const char *overlay_size = cmdline_get_or_default(cmd, "overlay_size", "1G");
    char tmpfs_opts[256];
    snprintf(tmpfs_opts, sizeof(tmpfs_opts), "size=%s", overlay_size);

    // Mount tmpfs for upper layer
    if (mount("tmpfs", "/upper", "tmpfs", 0, tmpfs_opts)) {
        LOG_BOOT("Failed to mount tmpfs upper: %s", strerror(errno));
        umount("/lower");
        return -1;
    }

    // CRITICAL: Create upper and work directories INSIDE the tmpfs mount
    // overlayfs requires workdir to be on same filesystem as upperdir
    mkdir("/upper/upper", 0755);
    mkdir("/upper/work", 0755);

    // Mount overlayfs
    char overlay_opts[512];
    snprintf(overlay_opts, sizeof(overlay_opts), "lowerdir=/lower,upperdir=/upper/upper,workdir=/upper/work");

    if (mount("overlay", "/newroot", "overlay", 0, overlay_opts)) {
        LOG_BOOT("Failed to mount overlayfs: %s", strerror(errno));
        umount("/upper");
        umount("/lower");
        return -1;
    }

    LOG_BOOT("Overlayfs mounted (squashfs lower + tmpfs upper %s)", overlay_size);
    return 0;
}

static int mount_rootfs_direct(const char *device, const char *fstype) {
    LOG_BOOT("Mounting %s filesystem", fstype);

    mkdir("/newroot", 0755);

    if (mount(device, "/newroot", fstype, 0, NULL)) {
        LOG_BOOT("Failed to mount %s: %s", fstype, strerror(errno));
        return -1;
    }

    LOG_BOOT("Mounted %s filesystem", fstype);
    return 0;
}

static void pivot_to_rootfs(void) {
    LOG_BOOT("Pivoting to rootfs");

    // Chroot into newroot
    if (chdir("/newroot")) panic("chdir(/newroot)");
    if (chroot(".")) panic("chroot(.)");
    if (chdir("/")) panic("chdir(/)");

    // Remount essential filesystems in new root
    mount_essential_filesystems();
}

static void bootstrap_rootfs(const cmdline_t *cmd) {
    const char *root_device = cmdline_get_or_default(cmd, "root", "/dev/vda");
    const char *root_fstype = cmdline_get_or_default(cmd, "rootfstype", "ext4");

    // Check if we need to mount rootfs
    const char *boot_mode = cmdline_get(cmd, "volant.boot");
    if (boot_mode && strcmp(boot_mode, "initramfs") == 0) {
        LOG_BOOT("Boot mode: initramfs (skipping rootfs mount)");
        return;
    }

    LOG_BOOT("Boot mode: auto (mounting rootfs device=%s fstype=%s)", root_device, root_fstype);

    if (wait_for_device(root_device, 10) != 0) {
        FATAL("Root device %s not found", root_device);
    }

    int mount_result;
    if (strcmp(root_fstype, "squashfs") == 0) {
        mount_result = mount_rootfs_squashfs(root_device, cmd);
    } else {
        mount_result = mount_rootfs_direct(root_device, root_fstype);
    }

    if (mount_result != 0) {
        FATAL("Failed to mount rootfs");
    }

    pivot_to_rootfs();
}

// ============================================================================
// MANIFEST PARSING (minimal implementation for exec workloads)
// ============================================================================

// Simple JSON string extractor (finds "key":"value" patterns)
static char* extract_json_string(const char *json, const char *key) {
    char search[256];
    snprintf(search, sizeof(search), "\"%s\"", key);

    const char *key_pos = strstr(json, search);
    if (!key_pos) return NULL;

    const char *colon = strchr(key_pos, ':');
    if (!colon) return NULL;

    const char *start = strchr(colon, '"');
    if (!start) return NULL;
    start++; // Skip opening quote

    const char *end = strchr(start, '"');
    if (!end) return NULL;

    size_t len = end - start;
    char *value = malloc(len + 1);
    if (!value) return NULL;

    memcpy(value, start, len);
    value[len] = '\0';
    return value;
}

// Extract array from JSON (finds "key":[...])
static char** extract_json_array(const char *json, const char *key, int *count) {
    *count = 0;

    char search[256];
    snprintf(search, sizeof(search), "\"%s\"", key);

    const char *key_pos = strstr(json, search);
    if (!key_pos) return NULL;

    const char *bracket_start = strchr(key_pos, '[');
    if (!bracket_start) return NULL;

    const char *bracket_end = strchr(bracket_start, ']');
    if (!bracket_end) return NULL;

    // Count array elements (count quotes)
    int quotes = 0;
    for (const char *p = bracket_start; p < bracket_end; p++) {
        if (*p == '"') quotes++;
    }
    int elements = quotes / 2;

    if (elements == 0) return NULL;

    char **array = calloc(elements + 1, sizeof(char*)); // NULL-terminated
    if (!array) return NULL;

    // Extract each string
    const char *pos = bracket_start + 1;
    for (int i = 0; i < elements; i++) {
        const char *start = strchr(pos, '"');
        if (!start) break;
        start++;

        const char *end = strchr(start, '"');
        if (!end) break;

        size_t len = end - start;
        array[i] = malloc(len + 1);
        if (array[i]) {
            memcpy(array[i], start, len);
            array[i][len] = '\0';
            (*count)++;
        }

        pos = end + 1;
    }

    return array;
}

static manifest_t* parse_manifest_simple(const char *json_str) {
    if (!json_str) return NULL;

    manifest_t *m = calloc(1, sizeof(manifest_t));
    if (!m) return NULL;

    // Extract basic fields
    m->name = extract_json_string(json_str, "name");
    m->version = extract_json_string(json_str, "version");
    m->runtime = extract_json_string(json_str, "runtime");

    // Extract workload type
    m->workload.type = extract_json_string(json_str, "type");
    if (!m->workload.type) {
        m->workload.type = xstrdup("exec"); // Default
    }

    // Extract entrypoint
    m->workload.entrypoint = extract_json_array(json_str, "entrypoint", &m->workload.entrypoint_count);

    // Extract base_url for HTTP workloads
    m->workload.base_url = extract_json_string(json_str, "base_url");

    // Extract workdir
    m->workload.workdir = extract_json_string(json_str, "workdir");

    LOG("Parsed manifest: name=%s type=%s entrypoint_count=%d",
        m->name ? m->name : "(none)",
        m->workload.type,
        m->workload.entrypoint_count);

    return m;
}

static manifest_t* fetch_manifest_from_api(const cmdline_t *cmd) {
    const char *api_host = cmdline_get(cmd, "volant.api_host");
    const char *api_port = cmdline_get(cmd, "volant.api_port");
    const char *image_name = cmdline_get(cmd, "volant.image");

    if (!api_host || !api_port || !image_name) {
        return NULL;
    }

    // Simple HTTP GET implementation using raw sockets
    int sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock < 0) {
        LOG("Failed to create socket for manifest fetch: %s", strerror(errno));
        return NULL;
    }

    struct sockaddr_in server_addr = {0};
    server_addr.sin_family = AF_INET;
    server_addr.sin_port = htons(atoi(api_port));

    if (inet_pton(AF_INET, api_host, &server_addr.sin_addr) <= 0) {
        LOG("Invalid API host address: %s", api_host);
        close(sock);
        return NULL;
    }

    if (connect(sock, (struct sockaddr*)&server_addr, sizeof(server_addr)) < 0) {
        LOG("Failed to connect to API server %s:%s: %s", api_host, api_port, strerror(errno));
        close(sock);
        return NULL;
    }

    // Send HTTP GET request
    char request[1024];
    snprintf(request, sizeof(request),
        "GET /api/v1/images/%s/manifest HTTP/1.1\r\n"
        "Host: %s:%s\r\n"
        "Connection: close\r\n"
        "\r\n",
        image_name, api_host, api_port);

    if (write(sock, request, strlen(request)) < 0) {
        LOG("Failed to send HTTP request: %s", strerror(errno));
        close(sock);
        return NULL;
    }

    // Read response
    char response[16384] = {0};
    ssize_t total = 0;
    ssize_t n;

    while ((n = read(sock, response + total, sizeof(response) - total - 1)) > 0) {
        total += n;
        if (total >= (ssize_t)sizeof(response) - 1) break;
    }
    close(sock);

    if (total == 0) {
        LOG("Empty response from API server");
        return NULL;
    }

    // Find JSON body (after \r\n\r\n)
    char *body = strstr(response, "\r\n\r\n");
    if (!body) {
        LOG("Invalid HTTP response format");
        return NULL;
    }
    body += 4; // Skip \r\n\r\n

    LOG("Fetched manifest from API (%zd bytes)", strlen(body));
    return parse_manifest_simple(body);
}

static unsigned char* gzip_decompress(unsigned char *compressed, size_t compressed_len, size_t *decompressed_len) {
    *decompressed_len = 0;

    // Allocate buffer for decompressed data (assume max 64KB for manifest)
    size_t buf_size = 65536;
    unsigned char *buffer = malloc(buf_size);
    if (!buffer) return NULL;

    z_stream stream = {0};
    stream.next_in = compressed;
    stream.avail_in = compressed_len;
    stream.next_out = buffer;
    stream.avail_out = buf_size;

    // Use inflateInit2 with windowBits=15+16 for gzip format
    if (inflateInit2(&stream, 15 + 16) != Z_OK) {
        free(buffer);
        return NULL;
    }

    int ret = inflate(&stream, Z_FINISH);
    inflateEnd(&stream);

    if (ret != Z_STREAM_END) {
        LOG("Gzip decompression failed: %d", ret);
        free(buffer);
        return NULL;
    }

    *decompressed_len = stream.total_out;
    buffer[*decompressed_len] = '\0'; // Null terminate for JSON parsing
    return buffer;
}

static manifest_t* resolve_manifest(const cmdline_t *cmd) {
    // Try volant.manifest from cmdline first
    const char *manifest_encoded = cmdline_get(cmd, "volant.manifest");
    if (manifest_encoded) {
        unsigned char *compressed = NULL;
        size_t compressed_len = base64_decode(manifest_encoded, &compressed);

        if (compressed_len > 0 && compressed) {
            // Decompress gzip
            size_t json_len = 0;
            unsigned char *manifest_json = gzip_decompress(compressed, compressed_len, &json_len);
            free(compressed);

            if (json_len > 0 && manifest_json) {
                LOG("Decompressed manifest: %zu bytes", json_len);
                manifest_t *m = parse_manifest_simple((char*)manifest_json);
                free(manifest_json);
                return m;
            } else {
                LOG("Failed to decompress manifest");
            }
        }
    }

    // Fallback: fetch from API
    return fetch_manifest_from_api(cmd);
}

// ============================================================================
// WORKLOAD SUPERVISION
// ============================================================================

static void start_workload(manifest_t *manifest) {
    if (!manifest || !manifest->workload.entrypoint || manifest->workload.entrypoint_count == 0) {
        LOG("No workload specified in manifest");
        return;
    }

    LOG("Starting workload: %s", manifest->workload.entrypoint[0]);

    pid_t pid = fork();
    if (pid < 0) {
        LOG("Failed to fork workload: %s", strerror(errno));
        return;
    }

    if (pid == 0) {
        // Child process: exec workload

        // Set up PATH environment
        setenv("PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", 1);

        // Set environment variables from manifest
        if (manifest->workload.env) {
            for (int i = 0; i < manifest->workload.env_count; i++) {
                putenv(manifest->workload.env[i]);
            }
        }

        // Change working directory
        if (manifest->workload.workdir) {
            if (chdir(manifest->workload.workdir) != 0) {
                fprintf(stderr, "Warning: failed to chdir to %s: %s\n",
                    manifest->workload.workdir, strerror(errno));
            }
        }

        // Execute workload
        execvp(manifest->workload.entrypoint[0], manifest->workload.entrypoint);

        // If we get here, exec failed
        fprintf(stderr, "Failed to exec workload %s: %s\n",
            manifest->workload.entrypoint[0], strerror(errno));
        exit(1);
    }

    // Parent: store PID
    g_state.workload_pid = pid;
    LOG("Workload started (pid=%d)", pid);
}

static void reap_children(void) {
    int status;
    pid_t pid;

    while ((pid = waitpid(-1, &status, WNOHANG)) > 0) {
        if (pid == g_state.workload_pid) {
            LOG("Workload process exited (pid=%d, status=%d)", pid, WEXITSTATUS(status));
            g_state.workload_pid = 0;

            // Restart workload if configured
            if (g_state.manifest && g_state.workload_restart_count < MAX_WORKLOAD_RESTARTS) {
                g_state.workload_restart_count++;
                LOG("Restarting workload (attempt %d/%d) in %d seconds",
                    g_state.workload_restart_count, MAX_WORKLOAD_RESTARTS, WORKLOAD_RESTART_DELAY_SEC);
                sleep(WORKLOAD_RESTART_DELAY_SEC);
                start_workload(g_state.manifest);
            } else {
                LOG("Workload restart limit reached or no manifest");
            }
        }
    }
}

// ============================================================================
// HTTP API SERVER (simplified - will expand for full reverse proxy)
// ============================================================================

static void handle_health_check(int client_fd) {
    char response[512];
    time_t uptime = time(NULL) - g_state.start_time;

    int len = snprintf(response, sizeof(response),
        "HTTP/1.1 200 OK\r\n"
        "Content-Type: application/json\r\n"
        "Connection: close\r\n"
        "\r\n"
        "{\"status\":\"ok\",\"version\":\"%s\",\"uptime\":%ld}\r\n",
        KESTREL_VERSION, uptime);

    ssize_t written = write(client_fd, response, len);
    (void)written; // Ignore write errors for health check
}

static void handle_http_request(int client_fd) {
    char buffer[4096];
    ssize_t n = read(client_fd, buffer, sizeof(buffer) - 1);

    if (n <= 0) {
        close(client_fd);
        return;
    }

    buffer[n] = '\0';

    // Parse request line
    if (strncmp(buffer, "GET /healthz", 12) == 0) {
        handle_health_check(client_fd);
    } else {
        // 404 for everything else (TODO: add reverse proxy)
        const char *response = "HTTP/1.1 404 Not Found\r\nConnection: close\r\n\r\n";
        ssize_t written = write(client_fd, response, strlen(response));
        (void)written; // Ignore write errors for 404
    }

    close(client_fd);
}

static int start_api_server(void) {
    int sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock < 0) {
        LOG("Failed to create socket: %s", strerror(errno));
        return -1;
    }

    int opt = 1;
    setsockopt(sock, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

    struct sockaddr_in addr = {0};
    addr.sin_family = AF_INET;
    addr.sin_addr.s_addr = INADDR_ANY;
    addr.sin_port = htons(API_PORT);

    if (bind(sock, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        LOG("Failed to bind to port %d: %s", API_PORT, strerror(errno));
        close(sock);
        return -1;
    }

    if (listen(sock, 128) < 0) {
        LOG("Failed to listen: %s", strerror(errno));
        close(sock);
        return -1;
    }

    LOG("HTTP API server listening on port %d", API_PORT);
    return sock;
}

// ============================================================================
// SIGNAL HANDLERS
// ============================================================================

static void sigchld_handler(int sig) {
    (void)sig;
    // Will be handled in main loop via waitpid
}

static void sigterm_handler(int sig) {
    (void)sig;
    LOG("Received SIGTERM, shutting down gracefully");
    g_state.should_exit = 1;
}

static void setup_signal_handlers(void) {
    signal(SIGCHLD, sigchld_handler);
    signal(SIGTERM, sigterm_handler);
    signal(SIGINT, sigterm_handler);
}

// ============================================================================
// MAIN
// ============================================================================

int main(int argc, char *argv[]) {
    (void)argc;
    (void)argv;

    g_state.start_time = time(NULL);

    // Phase 1: Early boot
    LOG_BOOT("Kestrel v%s starting (PID %d)", KESTREL_VERSION, getpid());

    // Create directory structure
    mkdir("/bin", 0755);
    mkdir("/sbin", 0755);
    mkdir("/usr", 0755);

    // Mount essential filesystems
    mount_essential_filesystems();
    ensure_console();

    LOG_BOOT("Essential filesystems mounted");

    // Phase 2: Parse kernel cmdline
    parse_cmdline(&g_state.cmdline);

    // Phase 3: Setup environment
    setup_environment_from_cmdline(&g_state.cmdline);
    setup_dns_from_cmdline(&g_state.cmdline);

    // Phase 4: Mount rootfs
    bootstrap_rootfs(&g_state.cmdline);

    // Phase 5: Resolve manifest
    LOG("Resolving workload manifest");
    g_state.manifest = resolve_manifest(&g_state.cmdline);

    if (!g_state.manifest) {
        LOG("No manifest found, running in idle mode");
    }

    // Phase 6: Start workload
    if (g_state.manifest) {
        start_workload(g_state.manifest);
    }

    // Phase 7: Start API server
    g_state.api_socket = start_api_server();
    if (g_state.api_socket < 0) {
        LOG("Warning: API server failed to start");
    }

    // Setup signal handlers
    setup_signal_handlers();

    LOG("Kestrel initialization complete, entering main loop");

    // Main event loop
    while (!g_state.should_exit) {
        fd_set read_fds;
        FD_ZERO(&read_fds);

        int max_fd = -1;

        if (g_state.api_socket >= 0) {
            FD_SET(g_state.api_socket, &read_fds);
            max_fd = g_state.api_socket;
        }

        struct timeval timeout = {1, 0}; // 1 second timeout

        int ready = select(max_fd + 1, &read_fds, NULL, NULL, &timeout);

        if (ready < 0) {
            if (errno == EINTR) continue;
            LOG("select() error: %s", strerror(errno));
            break;
        }

        if (ready > 0 && g_state.api_socket >= 0 && FD_ISSET(g_state.api_socket, &read_fds)) {
            struct sockaddr_in client_addr;
            socklen_t client_len = sizeof(client_addr);

            int client_fd = accept(g_state.api_socket, (struct sockaddr*)&client_addr, &client_len);
            if (client_fd >= 0) {
                handle_http_request(client_fd);
            }
        }

        // Reap child processes
        reap_children();
    }

    LOG("Kestrel shutting down");

    // Cleanup
    if (g_state.api_socket >= 0) {
        close(g_state.api_socket);
    }

    if (getpid() == 1) {
        LOG("PID 1 cannot exit, halting system");
        reboot(RB_POWER_OFF);
    }

    return 0;
}

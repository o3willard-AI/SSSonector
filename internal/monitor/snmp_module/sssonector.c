#include <net-snmp/net-snmp-config.h>
#include <net-snmp/net-snmp-includes.h>
#include <net-snmp/agent/net-snmp-agent-includes.h>
#include <signal.h>
#include <sys/ipc.h>
#include <sys/shm.h>
#include <time.h>

// Base OID for SSL-TUNNEL-MIB (enterprises.2021.10)
static oid sssonector_oid[] = {1,3,6,1,4,1,2021,54321};

// Stats OIDs (enterprises.2021.10.1.3.X)
static oid bytes_received_oid[] = {1,3,6,1,4,1,2021,54321,1,3,1};
static oid bytes_sent_oid[] = {1,3,6,1,4,1,2021,54321,1,3,2};
static oid packets_lost_oid[] = {1,3,6,1,4,1,2021,54321,1,3,3};
static oid latency_oid[] = {1,3,6,1,4,1,2021,54321,1,3,4};
static oid uptime_oid[] = {1,3,6,1,4,1,2021,54321,1,3,5};
static oid cpu_usage_oid[] = {1,3,6,1,4,1,2021,54321,1,3,6};
static oid memory_usage_oid[] = {1,3,6,1,4,1,2021,54321,1,3,7};
static oid active_connections_oid[] = {1,3,6,1,4,1,2021,54321,1,3,8};
static oid total_connections_oid[] = {1,3,6,1,4,1,2021,54321,1,3,9};

// Shared memory key for metrics
#define SHM_KEY 0x534E4D50  // "SNMP" in hex

// Metrics structure matching MIB
typedef struct {
    uint64_t bytes_received;
    uint64_t bytes_sent;
    uint64_t packets_lost;
    int32_t latency;          // microseconds
    int32_t uptime;           // seconds
    char cpu_usage[32];       // DisplayString
    char memory_usage[32];    // DisplayString
    uint32_t active_connections;
    uint64_t total_connections;
    time_t start_time;
} metrics_t;

static metrics_t *metrics = NULL;

// Function to initialize shared memory
static int init_shared_memory(void) {
    int shmid;
    
    if ((shmid = shmget(SHM_KEY, sizeof(metrics_t), IPC_CREAT | 0666)) < 0) {
        snmp_log(LOG_ERR, "shmget failed\n");
        return 0;
    }
    
    if ((metrics = (metrics_t *)shmat(shmid, NULL, 0)) == (metrics_t *)-1) {
        snmp_log(LOG_ERR, "shmat failed\n");
        return 0;
    }

    // Initialize with test data
    metrics->bytes_received = 22598313;
    metrics->bytes_sent = 6658912;
    metrics->packets_lost = 0;
    metrics->latency = 45200;  // 45.2ms in microseconds
    metrics->start_time = time(NULL);
    metrics->uptime = 0;  // Will be calculated on each request
    snprintf(metrics->cpu_usage, sizeof(metrics->cpu_usage), "25%%");
    snprintf(metrics->memory_usage, sizeof(metrics->memory_usage), "512MB");
    metrics->active_connections = 5;
    metrics->total_connections = 42;
    
    snmp_log(LOG_INFO, "Shared memory initialized with test data\n");
    return 1;
}

// Generic handler for Counter64 metrics
static int handle_counter64(uint64_t value, netsnmp_request_info *requests) {
    struct counter64 c64;
    c64.high = value >> 32;
    c64.low = value & 0xFFFFFFFF;
    snmp_set_var_typed_value(requests->requestvb, ASN_COUNTER64,
                            (u_char*)&c64, sizeof(c64));
    return SNMP_ERR_NOERROR;
}

// Handlers for individual metrics
static int handle_bytes_received(netsnmp_mib_handler *handler,
                               netsnmp_handler_registration *reginfo,
                               netsnmp_agent_request_info *reqinfo,
                               netsnmp_request_info *requests) {
    snmp_log(LOG_INFO, "Handling bytes_received request\n");
    if (!metrics) return SNMP_ERR_GENERR;
    if (reqinfo->mode == MODE_GET) {
        return handle_counter64(metrics->bytes_received, requests);
    }
    return SNMP_ERR_GENERR;
}

static int handle_bytes_sent(netsnmp_mib_handler *handler,
                           netsnmp_handler_registration *reginfo,
                           netsnmp_agent_request_info *reqinfo,
                           netsnmp_request_info *requests) {
    snmp_log(LOG_INFO, "Handling bytes_sent request\n");
    if (!metrics) return SNMP_ERR_GENERR;
    if (reqinfo->mode == MODE_GET) {
        return handle_counter64(metrics->bytes_sent, requests);
    }
    return SNMP_ERR_GENERR;
}

static int handle_packets_lost(netsnmp_mib_handler *handler,
                             netsnmp_handler_registration *reginfo,
                             netsnmp_agent_request_info *reqinfo,
                             netsnmp_request_info *requests) {
    snmp_log(LOG_INFO, "Handling packets_lost request\n");
    if (!metrics) return SNMP_ERR_GENERR;
    if (reqinfo->mode == MODE_GET) {
        return handle_counter64(metrics->packets_lost, requests);
    }
    return SNMP_ERR_GENERR;
}

static int handle_latency(netsnmp_mib_handler *handler,
                         netsnmp_handler_registration *reginfo,
                         netsnmp_agent_request_info *reqinfo,
                         netsnmp_request_info *requests) {
    snmp_log(LOG_INFO, "Handling latency request\n");
    if (!metrics) return SNMP_ERR_GENERR;
    if (reqinfo->mode == MODE_GET) {
        snmp_set_var_typed_value(requests->requestvb, ASN_INTEGER,
                               (u_char*)&metrics->latency,
                               sizeof(metrics->latency));
        return SNMP_ERR_NOERROR;
    }
    return SNMP_ERR_GENERR;
}

static int handle_uptime(netsnmp_mib_handler *handler,
                        netsnmp_handler_registration *reginfo,
                        netsnmp_agent_request_info *reqinfo,
                        netsnmp_request_info *requests) {
    snmp_log(LOG_INFO, "Handling uptime request\n");
    if (!metrics) return SNMP_ERR_GENERR;
    if (reqinfo->mode == MODE_GET) {
        metrics->uptime = time(NULL) - metrics->start_time;
        snmp_set_var_typed_value(requests->requestvb, ASN_INTEGER,
                               (u_char*)&metrics->uptime,
                               sizeof(metrics->uptime));
        return SNMP_ERR_NOERROR;
    }
    return SNMP_ERR_GENERR;
}

static int handle_cpu_usage(netsnmp_mib_handler *handler,
                          netsnmp_handler_registration *reginfo,
                          netsnmp_agent_request_info *reqinfo,
                          netsnmp_request_info *requests) {
    snmp_log(LOG_INFO, "Handling cpu_usage request\n");
    if (!metrics) return SNMP_ERR_GENERR;
    if (reqinfo->mode == MODE_GET) {
        snmp_set_var_typed_value(requests->requestvb, ASN_OCTET_STR,
                               (u_char*)metrics->cpu_usage,
                               strlen(metrics->cpu_usage));
        return SNMP_ERR_NOERROR;
    }
    return SNMP_ERR_GENERR;
}

static int handle_memory_usage(netsnmp_mib_handler *handler,
                             netsnmp_handler_registration *reginfo,
                             netsnmp_agent_request_info *reqinfo,
                             netsnmp_request_info *requests) {
    snmp_log(LOG_INFO, "Handling memory_usage request\n");
    if (!metrics) return SNMP_ERR_GENERR;
    if (reqinfo->mode == MODE_GET) {
        snmp_set_var_typed_value(requests->requestvb, ASN_OCTET_STR,
                               (u_char*)metrics->memory_usage,
                               strlen(metrics->memory_usage));
        return SNMP_ERR_NOERROR;
    }
    return SNMP_ERR_GENERR;
}

static int handle_active_connections(netsnmp_mib_handler *handler,
                                   netsnmp_handler_registration *reginfo,
                                   netsnmp_agent_request_info *reqinfo,
                                   netsnmp_request_info *requests) {
    snmp_log(LOG_INFO, "Handling active_connections request\n");
    if (!metrics) return SNMP_ERR_GENERR;
    if (reqinfo->mode == MODE_GET) {
        snmp_set_var_typed_value(requests->requestvb, ASN_GAUGE,
                               (u_char*)&metrics->active_connections,
                               sizeof(metrics->active_connections));
        return SNMP_ERR_NOERROR;
    }
    return SNMP_ERR_GENERR;
}

static int handle_total_connections(netsnmp_mib_handler *handler,
                                  netsnmp_handler_registration *reginfo,
                                  netsnmp_agent_request_info *reqinfo,
                                  netsnmp_request_info *requests) {
    snmp_log(LOG_INFO, "Handling total_connections request\n");
    if (!metrics) return SNMP_ERR_GENERR;
    if (reqinfo->mode == MODE_GET) {
        return handle_counter64(metrics->total_connections, requests);
    }
    return SNMP_ERR_GENERR;
}

// Initialize the module
void init_sssonector(void) {
    snmp_log(LOG_INFO, "Initializing SSonector SNMP Module\n");
    
    if (!init_shared_memory()) {
        snmp_log(LOG_ERR, "Failed to initialize shared memory\n");
        return;
    }

    // Register all metrics
    netsnmp_handler_registration *reg;

    reg = netsnmp_create_handler_registration(
        "bytesReceived", handle_bytes_received,
        bytes_received_oid, OID_LENGTH(bytes_received_oid),
        HANDLER_CAN_RONLY);
    if (netsnmp_register_scalar(reg) != MIB_REGISTERED_OK)
        snmp_log(LOG_ERR, "Failed to register bytesReceived\n");

    reg = netsnmp_create_handler_registration(
        "bytesSent", handle_bytes_sent,
        bytes_sent_oid, OID_LENGTH(bytes_sent_oid),
        HANDLER_CAN_RONLY);
    if (netsnmp_register_scalar(reg) != MIB_REGISTERED_OK)
        snmp_log(LOG_ERR, "Failed to register bytesSent\n");

    reg = netsnmp_create_handler_registration(
        "packetsLost", handle_packets_lost,
        packets_lost_oid, OID_LENGTH(packets_lost_oid),
        HANDLER_CAN_RONLY);
    if (netsnmp_register_scalar(reg) != MIB_REGISTERED_OK)
        snmp_log(LOG_ERR, "Failed to register packetsLost\n");

    reg = netsnmp_create_handler_registration(
        "latency", handle_latency,
        latency_oid, OID_LENGTH(latency_oid),
        HANDLER_CAN_RONLY);
    if (netsnmp_register_scalar(reg) != MIB_REGISTERED_OK)
        snmp_log(LOG_ERR, "Failed to register latency\n");

    reg = netsnmp_create_handler_registration(
        "uptime", handle_uptime,
        uptime_oid, OID_LENGTH(uptime_oid),
        HANDLER_CAN_RONLY);
    if (netsnmp_register_scalar(reg) != MIB_REGISTERED_OK)
        snmp_log(LOG_ERR, "Failed to register uptime\n");

    reg = netsnmp_create_handler_registration(
        "cpuUsage", handle_cpu_usage,
        cpu_usage_oid, OID_LENGTH(cpu_usage_oid),
        HANDLER_CAN_RONLY);
    if (netsnmp_register_scalar(reg) != MIB_REGISTERED_OK)
        snmp_log(LOG_ERR, "Failed to register cpuUsage\n");

    reg = netsnmp_create_handler_registration(
        "memoryUsage", handle_memory_usage,
        memory_usage_oid, OID_LENGTH(memory_usage_oid),
        HANDLER_CAN_RONLY);
    if (netsnmp_register_scalar(reg) != MIB_REGISTERED_OK)
        snmp_log(LOG_ERR, "Failed to register memoryUsage\n");

    reg = netsnmp_create_handler_registration(
        "activeConnections", handle_active_connections,
        active_connections_oid, OID_LENGTH(active_connections_oid),
        HANDLER_CAN_RONLY);
    if (netsnmp_register_scalar(reg) != MIB_REGISTERED_OK)
        snmp_log(LOG_ERR, "Failed to register activeConnections\n");

    reg = netsnmp_create_handler_registration(
        "totalConnections", handle_total_connections,
        total_connections_oid, OID_LENGTH(total_connections_oid),
        HANDLER_CAN_RONLY);
    if (netsnmp_register_scalar(reg) != MIB_REGISTERED_OK)
        snmp_log(LOG_ERR, "Failed to register totalConnections\n");

    snmp_log(LOG_INFO, "SSonector SNMP Module initialization complete\n");
}

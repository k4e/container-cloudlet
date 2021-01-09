#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <errno.h>
#include <sys/time.h>
#include <unistd.h>
#include <sys/socket.h>
#include <arpa/inet.h>

#define SV_PORT 8888
#define BUF_SIZE (8*1024*1024+1)
#define MAX_CONNECTION 8
#define TIMEOUT_SEC 15

int g_upstream_mode = 0;
volatile char g_buf[BUF_SIZE];

void start_echo(int svsock);
void echo_routine(int clisock);

void start_echo(int svsock) {
    pid_t pid;
    int clisock;
    struct sockaddr_in cliaddr;
    socklen_t cliaddr_len;

    while (1) {
        cliaddr_len = sizeof(cliaddr);
        clisock = accept(svsock, (struct sockaddr *)&cliaddr, &cliaddr_len);
        if (clisock != -1) {
#ifdef DEBUG
            fprintf(stdout, "Accept: %s:%d\r\n", inet_ntoa(cliaddr.sin_addr),
                    ntohs(cliaddr.sin_port));
#endif
            pid = fork();
            if (pid < 0) {
#ifdef DEBUG
                fprintf(stdout, "Error: fork: %d\r\n", errno);
#endif
                close(clisock);
            } else if (pid == 0) {
                echo_routine(clisock);
                close(clisock);
#ifdef DEBUG
                fprintf(stdout, "Close: %s:%d\r\n", inet_ntoa(cliaddr.sin_addr),
                        ntohs(cliaddr.sin_port));
#endif
                exit(0);
            } else {
                close(clisock);
            }
        } else {
#ifdef DEBUG
            fprintf(stdout, "Error: accept: %d\r\n", errno);
#endif
            return;
        }
    }
}

void echo_routine(int clisock) {
    ssize_t recvsz;
    ssize_t sentsz;
    int upstream_mode;
    char buf_1b[1];
    int i;
    int s;
    while (1) {
        recvsz = read(clisock, (char *)g_buf, BUF_SIZE);
        if (recvsz == -1) {
#ifdef DEBUG
            fprintf(stdout, "Error: read: %d\r\n", errno);
#endif
            return;
        }
        sentsz = 0;
        upstream_mode = 0;
        if (recvsz > 0 && g_buf[0] == 'U') {
            upstream_mode = 1;
        }
        if (upstream_mode) {
            s = 0;
            for (i = 0; i < recvsz; ++i) {
                s = (s + g_buf[i]) % 128;
            }
            buf_1b[0] = (char)s;
            sentsz = write(clisock, buf_1b, 1);
        } else {
            sentsz = write(clisock, (char *)g_buf, recvsz);
        }
        if (sentsz == -1) {
#ifdef DEBUG
            fprintf(stdout, "Error: write: %d\r\n", errno);
#endif
            return;
        }
    }
}

int main(int argc, char *argv[]) {
    int i;
    struct sockaddr_in svaddr;
    int svsock;
    int itrue = 1;
    char *env_direction;
    char *env_padding;
    int padsz_mb = 0;
    int padsz;
    volatile char *pad_mem;
    struct timeval tv_timeout;
    env_direction = getenv("DIRECTION");
    env_padding = getenv("PADDING");
    if (env_direction != NULL) {
        if (strcmp(env_direction, "UP") == 0) {
            g_upstream_mode = 1;
        }
    }
    if (env_padding != NULL) {
        padsz_mb = atoi(env_padding);
        if (0 < padsz_mb) {
            padsz = padsz_mb * 1024 * 1024;
            pad_mem = (char *)calloc(padsz, sizeof(char));
            memset((void *)pad_mem, '*', padsz);
        }
    }
    fprintf(stdout, "Build C 2012-01-06.3 (Upstream-mode=%d, Padding=%dMiB)\r\n",
            g_upstream_mode, padsz_mb);
    while (1) {
        memset(&svaddr, 0, sizeof(svaddr));
        svaddr.sin_port = htons(SV_PORT);
        svaddr.sin_family = AF_INET;
        svaddr.sin_addr.s_addr = htonl(INADDR_ANY);
#ifdef DEBUG
        fprintf(stdout, "Socket open\r\n");
#endif
        svsock = socket(AF_INET, SOCK_STREAM, 0);
        if(setsockopt(svsock, SOL_SOCKET, SO_REUSEADDR, &itrue, sizeof(itrue)) == -1) {
#ifdef DEBUG

#endif
            close(svsock);
        }
        bind(svsock, (struct sockaddr *)&svaddr, sizeof(svaddr));
        if (listen(svsock, MAX_CONNECTION) == -1) {
#ifdef DEBUG
            fprintf(stdout, "Error: listen: %d\r\n", errno);
#endif
            close(svsock);
        }
        start_echo(svsock);
        close(svsock);
    }
    return 0;
}

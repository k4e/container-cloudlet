package com.github.k4e;

import java.io.IOException;
import java.net.Socket;

public class SocketClient {
    
    public static SocketClient of(String host, int port, SocketProc proc) {
        return new SocketClient(host, port, proc);
    }

    private final String host;
    private final int port;
    private final SocketProc proc;

    private SocketClient(String host, int port, SocketProc proc) {
        this.host = host;
        this.port = port;
        this.proc = proc;
    }

    public void conn() throws IOException {
        Socket sock = null;
        try {
            sock = new Socket(host, port);
            System.out.println("Connection open");
            proc.accept(sock);
        } finally {
            if (sock != null) {
                sock.close();
                System.out.println("Connection closed");
            }
        }
    }
}

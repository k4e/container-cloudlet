package com.github.k4e;

import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintWriter;
import java.net.Socket;

import com.google.common.base.Strings;

public class SocketClient {

    private String host;
    private int port;
    private String msg;

    public SocketClient(String host, int port, String msg) {
        this.host = host;
        this.port = port;
        this.msg = msg;
    }

    public void exec() throws IOException {
        Socket sock = null;
        try {
            sock = new Socket(host, port);
            System.out.println("Connection open");
            accept(sock);
        } finally {
            if (sock != null) {
                sock.close();
                System.out.println("Connection closed");
            }
        }
    }

    public void accept(Socket sock) throws IOException {
        boolean msgIsEmpty = Strings.isNullOrEmpty(msg);
        if (!msgIsEmpty) {
            PrintWriter writer = new PrintWriter(sock.getOutputStream());
            writer.println(msg);
            writer.flush();
        }
        System.out.printf("Sent: %s\n", !msgIsEmpty ? msg : "(none)");
        char[] buf = new char[4096];
        InputStreamReader reader = new InputStreamReader(sock.getInputStream());
        int count = reader.read(buf);
        System.out.printf("Recv: %s\n", count > 0 ? new String(buf, 0, count) : "(none)");
    }
}

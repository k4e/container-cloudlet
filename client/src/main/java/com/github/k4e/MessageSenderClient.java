package com.github.k4e;

import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintWriter;
import java.net.Socket;

import com.github.k4e.types.ProtocolHeader;
import com.google.common.base.Strings;

public class MessageSenderClient {

    public static MessageSenderClient ofDefault() {
        return new MessageSenderClient(null);
    }

    public static MessageSenderClient ofSession(ProtocolHeader header) {
        return new MessageSenderClient(header);
    }

    private final ProtocolHeader header;

    private MessageSenderClient(ProtocolHeader header) {
        this.header = header;
    }

    public void send(String host, int port, String msg) throws IOException {
        Socket sock = null;
        char[] buf = new char[4096];
        try {
            sock = new Socket(host, port);
            System.out.println("Connection open");
            if (header != null) {
                writeHeader(sock);
            }
            if (!Strings.isNullOrEmpty(msg)) {
                PrintWriter writer = new PrintWriter(sock.getOutputStream());
                writer.println(msg);
                writer.flush();
                System.out.println("Sent: " + msg);
            } else {
                System.out.println("Sent none");
            }
            InputStreamReader reader = new InputStreamReader(sock.getInputStream());
            int count = reader.read(buf);
            if (count > 0) {
                System.out.println("Recv: " + new String(buf, 0, count));
            } else {
                System.out.println("Recv none");
            }
        } finally {
            if (sock != null) {
                sock.close();
                System.out.println("Connection closed");
            }
        }
    }

    private void writeHeader(Socket sock) throws IOException {
        System.out.println("Header: " + header.toString());
        byte[] b = header.getBytes();
        sock.getOutputStream().write(b);
        sock.getOutputStream().flush();
    }
}

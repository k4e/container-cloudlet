package com.github.k4e;

import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintWriter;
import java.net.Socket;

public class MessageSenderClient {

    public static void send(String host, int port, String msg) throws IOException {
        Socket sock = null;
        char[] buf = new char[4096];
        try {
            sock = new Socket(host, port);
            PrintWriter writer = new PrintWriter(sock.getOutputStream());
            writer.println(msg);
            writer.flush();
            System.out.println("Sent: " + msg);
            InputStreamReader reader = new InputStreamReader(sock.getInputStream());
            int count = reader.read(buf);
            System.out.print("Recv: ");
            if (count > 0) {
                System.out.println(new String(buf, 0, count));
            } else {
                System.out.println("");
            }
        } finally {
            if (sock != null) {
                sock.close();
            }
        }
    }
}

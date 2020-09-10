package com.github.k4e;

import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintWriter;
import java.net.Socket;

import com.google.common.base.Strings;

public class SocketSendRecv {

    public static void sendRecv(String host, int port, String msg) throws IOException {
        SocketClient.of(host, port, new SendRecvProc(msg)).conn();
    }

    static class SendRecvProc implements SocketProc {
        private String msg;
        SendRecvProc(String msg) {
            this.msg = msg;
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
}

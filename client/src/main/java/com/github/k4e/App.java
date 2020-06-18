package com.github.k4e;

import java.io.IOException;

public class App {
    public static void main(String[] args) throws Exception {
        if (args.length < 3) {
            System.err.println("Host, port, and operation must be given");
            System.exit(-1);
        }
        String host = args[0];
        int port = Integer.parseInt(args[1]);
        String op = args[2];
        switch (op) {
        case "send":
            String msg = args.length >= 4 ? args[3] : "";
            send(host, port, msg);
            break;
        case "create":
            create(host, port);
            break;
        case "delete":
            delete(host, port);
            break;
        default:
            throw new UnsupportedOperationException(op);
        }
    }

    private static void send(String host, int port, String msg) throws IOException {
        MessageSenderClient.send(host, port, msg);
    }

    private static void create(String host, int port) throws IOException {
        CloudletClient.create(host, port);
    }

    private static void delete(String host, int port) throws IOException {
        CloudletClient.delete(host, port);
    }
}

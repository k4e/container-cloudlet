package com.github.k4e;

import java.io.IOException;
import java.util.Arrays;
import java.util.UUID;

public class App {

    public static final UUID SESH_UUID = UUID.fromString("55C497AC-8AD5-4DA1-8673-6199443AE137");

    public static void main(String[] args) throws Exception {
        if (args.length < 3) {
            System.err.println("Host, port, and operation must be given");
            System.exit(-1);
        }
        String host = args[0];
        int port = Integer.parseInt(args[1]);
        String op = args[2];
        String[] opArgs = Arrays.copyOfRange(args, 3, args.length);
        switch (op) {
        case "send":
            send(host, port, opArgs);
            break;
        case "create":
            create(host, port);
            break;
        case "delete":
            delete(host, port);
            break;
        case "session":
        case "sesh":
            session(host, port, opArgs);
            break;
        default:
            throw new UnsupportedOperationException(op);
        }
    }

    private static void send(String host, int port, String[] opArgs) throws IOException {
        String msg = (opArgs.length > 0 ? opArgs[0] : null);
        MessageSenderClient.ofDefault().send(host, port, msg);
    }

    private static void create(String host, int port) throws IOException {
        CloudletClient.create(host, port);
    }

    private static void delete(String host, int port) throws IOException {
        CloudletClient.delete(host, port);
    }

    private static void session(String host, int port, String[] opArgs) throws IOException {
        String hostIP = null;
        boolean resume = false;
        String msg = null;
        for (int i = 0; i < opArgs.length; ++i) {
            String arg = opArgs[i];
            if ("-f".equals(arg) || "--forward".equals(arg)) {
                if (i + 1 < opArgs.length) {
                    hostIP = opArgs[i+1];
                    ++i;
                } else {
                    System.err.println("--forward must be followed by hostIP");
                    System.exit(-1);
                }
            } else if ("-r".equals(arg) || "--resume".equals("arg")) {
                resume = true;
            } else {
                if (msg == null) {
                    msg = arg;
                }
            }
        }
        CloudletClient.session(host, port, msg, SESH_UUID, hostIP, resume);
    }
}

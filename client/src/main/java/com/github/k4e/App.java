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

    private static void create(String host, int port) throws IOException {
        CloudletClient.create(host, port);
    }

    private static void delete(String host, int port) throws IOException {
        CloudletClient.delete(host, port);
    }

    private static void session(String host, int port, String[] methArgs) throws IOException {
        String fwdHostIp = null;
        boolean resume = false;
        for (int i = 0; i < methArgs.length; ++i) {
            String arg = methArgs[i];
            if ("-f".equals(arg) || "--forward".equals(arg)) {
                if (i + 1 < methArgs.length) {
                    fwdHostIp = methArgs[i+1];
                    ++i;
                } else {
                    System.err.println("--forward must be followed by hostIP");
                    System.exit(-1);
                }
            } else if ("-r".equals(arg) || "--resume".equals("arg")) {
                resume = true;
            }
        }
        SessionClient.of(host, port, SESH_UUID, fwdHostIp, resume).exec();
    }
}

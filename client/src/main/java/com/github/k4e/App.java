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
        String[] methArgs = Arrays.copyOfRange(args, 3, args.length);
        switch (op) {
        case "create":
            create(host, port, methArgs);
            break;
        case "delete":
            delete(host, port);
            break;
        case "session":
        case "sesh":
            session(host, port, methArgs);
            break;
        default:
            throw new UnsupportedOperationException(op);
        }
    }

    private static void create(String host, int port, String[] methArgs) throws IOException {
        boolean onlyFwd = false;
        for (int i = 0; i < methArgs.length; ++i) {
            String arg = methArgs[i];
            if ("-o".equals(arg) || "--only-forward".equals(arg)) {
                onlyFwd = true;
            } else {
                System.err.println("Warning: ignored arg: " + arg);
            }
        }
        CloudletClient.create(host, port, !onlyFwd);
    }

    private static void delete(String host, int port) throws IOException {
        CloudletClient.delete(host, port);
    }

    private static void session(String host, int port, String[] methArgs) throws IOException {
        String fwdHost = null;
        boolean resume = false;
        for (int i = 0; i < methArgs.length; ++i) {
            String arg = methArgs[i];
            if ("-f".equals(arg) || "--forward".equals(arg)) {
                if (i + 1 < methArgs.length) {
                    fwdHost = methArgs[i+1];
                    ++i;
                } else {
                    System.err.println("--forward requires hostIP");
                    System.exit(-1);
                }
            } else if ("-r".equals(arg) || "--resume".equals(arg)) {
                resume = true;
            } else {
                System.err.println("Warning: ignored arg: " + arg);
            }
        }
        String fwdHostIp = null;
        short fwdHostPort = 0;
        if (fwdHost != null) {
            try {
                String[] ipPort = fwdHost.split(":");
                if (ipPort.length != 2) {
                    throw new IllegalArgumentException();
                }
                fwdHostIp = ipPort[0];
                fwdHostPort = Short.parseShort(ipPort[1]);
            } catch (IllegalArgumentException e) {
                System.err.println("hostIp is expected as ipaddr:port but was %s" + fwdHost);
                System.exit(-1);
            }
        }
        SessionClient.of(host, port, SESH_UUID, fwdHostIp, fwdHostPort, resume).exec();
    }
}

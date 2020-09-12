package com.github.k4e;

import java.io.IOException;
import java.util.UUID;

public class App {

    public static final UUID SESH_UUID = UUID.fromString("55C497AC-8AD5-4DA1-8673-6199443AE137");

    public static void main(String[] args) throws Exception {
        if (args.length < 1) {
            System.err.println("Method must be given");
            System.exit(-1);
        }
        String meth = args[0];
        switch (meth) {
        case "create":
            create(args);
            break;
        case "delete":
            delete(args);
            break;
        case "session":
        case "sesh":
            session(args);
            break;
        case "experiment":
            experiment(args);
            break;
        default:
            throw new UnsupportedOperationException(meth);
        }
    }

    private static void create(String[] args) throws IOException {
        if (args.length < 3) {
            System.err.println("Host and port must be given");
            System.exit(-1);
        }
        String host = args[1];
        int port = Integer.parseInt(args[2]);
        boolean onlyFwd = false;
        for (int i = 3; i < args.length; ++i) {
            String arg = args[i];
            if ("-o".equals(arg) || "--only-forward".equals(arg)) {
                onlyFwd = true;
            } else {
                System.err.println("Warning: ignored arg: " + arg);
            }
        }
        CloudletClient.create(host, port, !onlyFwd);
    }

    private static void delete(String[] args) throws IOException {
        if (args.length < 3) {
            System.err.println("Host and port must be given");
            System.exit(-1);
        }
        String host = args[1];
        int port = Integer.parseInt(args[2]);
        CloudletClient.delete(host, port);
    }

    private static void session(String[] args) throws IOException {
        if (args.length < 3) {
            System.err.println("Host and port must be given");
            System.exit(-1);
        }
        String host = args[1];
        int port = Integer.parseInt(args[2]);
        String fwdHost = null;
        boolean resume = false;
        for (int i = 3; i < args.length; ++i) {
            String arg = args[i];
            if ("-f".equals(arg) || "--forward".equals(arg)) {
                if (i + 1 < args.length) {
                    fwdHost = args[i+1];
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

    public static void experiment(String[] args) throws IOException {
        if (args.length < 5) {
            System.err.println("Host-A, port-A, host-B, port-B must be given");
            System.exit(-1);
        }
        String hostA = args[1];
        int portA = Integer.parseInt(args[2]);
        String hostB = args[3];
        int portB = Integer.parseInt(args[4]);
        new Experiment(SESH_UUID).exec(hostA, portA, hostB, portB);
    }
}

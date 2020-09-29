package com.github.k4e;

import java.io.IOException;
import java.util.UUID;

import com.github.k4e.exp.CreationTest;
import com.github.k4e.exp.SpeedTest;

public class App {

    public static final UUID SESH_UUID = UUID.fromString("55C497AC-8AD5-4DA1-8673-6199443AE137");

    public static void main(String[] args) throws Exception {
        System.out.println("Build 2020-09-29");
        if (args.length < 1) {
            System.err.println("Method required");
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
            System.err.println("Required args: create <Host> <Port>");
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
            System.err.println("Required args: delete <Host> <Port>");
            System.exit(-1);
        }
        String host = args[1];
        int port = Integer.parseInt(args[2]);
        CloudletClient.delete(host, port);
    }

    private static void session(String[] args) throws IOException {
        if (args.length < 3) {
            System.err.println("Required args: session <Host> <Port>");
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
        if (args.length < 2) {
            System.err.println("Experiment case required");
            System.exit(-1);
        }
        String expCase = args[1];
        switch (expCase) {
        case "creation":
        case "create":
            creationTest(args);
            break;
        case "speed":
            speedTest(args);
            break;
        default:
            throw new UnsupportedOperationException(expCase);
        }
    }

    public static void creationTest(String[] args) throws IOException {
        if (args.length < 5) {
            System.err.println(
                "Required args: experiment creation <Host> <Cloudlet-port> <Session-port> <Case>");
            System.exit(-1);
        }
        String host = args[2];
        int cloudletPort = Integer.parseInt(args[3]);
        int sessionPort = Integer.parseInt(args[4]);
        String creatCase = args[5];
        boolean createApp;
        String fwdHost = null;
        short fwdPort = 0;
        switch (creatCase) {
        case "app":
        case "application":
            createApp = true;
            break;
        case "fwd":
        case "forward":
            createApp = false;
            if (args.length < 8) {
                System.err.println("forward requires host and port");
                System.exit(-1);
            }
            fwdHost = args[6];
            fwdPort = Short.parseShort(args[7]);
            break;
        default:
            throw new UnsupportedOperationException(creatCase);
        }
        new CreationTest(SESH_UUID).exec(host, cloudletPort, sessionPort, createApp, fwdHost, fwdPort);
    }

    public static void speedTest(String[] args) throws IOException {
        if (args.length < 7) {
            System.err.println("Required args: experiment speed <HostA> <PortA> <HostB> <PortB> <DataSize(KB)>");
            System.exit(-1);
        }
        String hostA = args[2];
        int portA = Integer.parseInt(args[3]);
        String hostB = args[4];
        int portB = Integer.parseInt(args[5]);
        int dataSize = Integer.parseInt(args[6]);
        int count = SpeedTest.DEFAULT_COUNT;
        boolean noWait = false;
        boolean fullCheck = false;
        for (int i = 1; i < args.length; ++i) {
            if ("-n".equals(args[i])) {
                if (i + 1 < args.length) {
                    count = Integer.parseInt(args[i + 1]);
                } else {
                    System.err.println("Count value required");
                    System.exit(-1);
                }
            } else if ("-W".equals(args[i])) {
                noWait = true;
            } else if ("-f".equals(args[i])) {
                fullCheck = true;
            }
        }
        new SpeedTest(SESH_UUID).exec(hostA, portA, hostB, portB, dataSize,
                count, noWait, fullCheck);
    }
}

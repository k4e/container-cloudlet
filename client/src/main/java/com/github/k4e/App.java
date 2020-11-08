package com.github.k4e;

import java.io.IOException;
import java.util.Arrays;
import java.util.stream.Collectors;

import com.github.k4e.exp.PerformanceTest;
import com.github.k4e.types.Request;

public class App {

    // public static final UUID SESH_UUID = UUID.fromString("55C497AC-8AD5-4DA1-8673-6199443AE137");

    public static void main(String[] args) throws Exception {
        System.out.println("Build 2020-11-04");
        if (args.length < 1) {
            System.err.println("Method required: [deploy|remove|send|session|experiment]");
            System.exit(-1);
        }
        String meth = args[0];
        switch (meth) {
        case "deploy":
            deploy(args);
            break;
        case "remove":
            remove(args);
            break;
        case "send":
            send(args);
            break;
        case "session":
        case "sesh":
            session(args);
            break;
        case "experiment":
            experiment(args);
            break;
        default:
            System.err.println("Unsupported method: " + meth);
            System.exit(-1);
        }
    }

    private static void deploy(String[] args) throws IOException {
        if (args.length < 3) {
            System.err.printf("Required args: %s %s <Host>\n", args[0], deployTypeDescription());
            System.exit(-1);
        }
        String type = args[1];
        String host = args[2];
        Request.Deploy.Type ctype = Request.Deploy.Type.valueOfIgnoreCase(type);
        if (ctype == null) {
            System.err.println("Unsupported deploy type: " + type);
            System.exit(-1);
        }
        String srcAddr = null;
        if (ctype == Request.Deploy.Type.LM
                || ctype == Request.Deploy.Type.FWD
                || ctype == Request.Deploy.Type.FWDLM) {
            if (args.length < 4) {
                System.err.println("Src addr is required");
                System.exit(-1);
            }
            srcAddr = args[3];
        }
        CloudletClient.deploy(host, ctype, srcAddr);
    }

    private static void remove(String[] args) throws IOException {
        if (args.length < 2) {
            System.err.printf("Required args: %s <Host>\n", args[0]);
            System.exit(-1);
        }
        String host = args[1];
        CloudletClient.remove(host);
    }

    private static void send(String[] args) throws IOException {
        if (args.length < 3) {
            System.err.printf("Required args: %s <Host> <Port> [Message]\n", args[0]);
            System.exit(-1);
        }
        String host = args[1];
        int port = Integer.parseInt(args[2]);
        StringBuffer buffer = new StringBuffer();
        for (int i = 3; i < args.length; ++i) {
            buffer.append(args[i]);
            if (i < args.length - 1) {
                buffer.append(" ");
            }
        }
        new SocketClient(host, port, buffer.toString()).exec();
    }

    private static void session(String[] args) throws IOException {
        if (args.length < 3) {
            System.err.printf("Required args: %s <Host> <Port>\n", args[0]);
            System.exit(-1);
        }
        String host = args[1];
        int port = Integer.parseInt(args[2]);
        SessionClient.of(host, port).exec();
    }

    public static void experiment(String[] args) throws Exception {
        if (args.length < 2) {
            System.err.println("Which item of experiment?");
            System.exit(-1);
        }
        String item = args[1];
        switch (item) {
        case "deploy":
            deployTest(args);
            break;
        case "latency":
        case "ltc":
            latencyTest(args);
            break;
        case "throughput":
        case "thru":
            throughputTest(args);
            break;
        default:
            System.err.println("Unsupported expriment item: " + item);
            System.exit(-1);
        }
    }

    public static void deployTest(String[] args) throws Exception {
        if (args.length < 4) {
            System.err.printf("Required args: %s %s %s <Host>\n",
                args[0], args[1], deployTypeDescription());
            System.exit(-1);
        }
        String type = args[2];
        String hostAddr = args[3];
        Request.Deploy.Type ctype = Request.Deploy.Type.valueOfIgnoreCase(type);
        if (ctype == null) {
            System.err.println("Unsupported deploy type: " + type);
            System.exit(-1);
        }
        String srcAddr = null;
        if (ctype == Request.Deploy.Type.LM
                || ctype == Request.Deploy.Type.FWD
                || ctype == Request.Deploy.Type.FWDLM) {
            if (args.length < 5) {
                System.err.println("Src addr is required");
                System.exit(-1);
            }
            srcAddr = args[4];
        }
        new PerformanceTest().deployTest(hostAddr, ctype, srcAddr);
    }

    public static void latencyTest(String[] args) throws Exception {
        if (args.length < 4) {
            System.err.printf("Required args: %s %s <Host> <DataSize(KB)>\n",
                args[0], args[1]);
            System.exit(-1);
        }
        String hostAddr = args[2];
        int dataSizeKB = Integer.parseInt(args[3]);
        int count = PerformanceTest.DEFAULT_COUNT;
        boolean fullCheck = false;
        for (int i = 1; i < args.length; ++i) {
            if ("-c".equals(args[i])) {
                if (i + 1 < args.length) {
                    count = Integer.parseInt(args[i + 1]);
                } else {
                    System.err.println("-c requires count");
                    System.exit(-1);
                }
            } else if ("-f".equals(args[i])) {
                fullCheck = true;
            } else if (args[i].startsWith("-")) {
                System.err.println("Ignored option: " + args[i]);
            }
        }
        new PerformanceTest().latencyTest(hostAddr, dataSizeKB, count, fullCheck);
    }

    public static void throughputTest(String[] args) throws Exception {
        if (args.length < 4) {
            System.err.printf("Required args: %s %s %s <Host>\n",
                args[0], args[1], deployTypeDescription());
            System.exit(-1);
        }
        String type = args[2];
        String hostAddr = args[3];
        Request.Deploy.Type ctype = Request.Deploy.Type.valueOfIgnoreCase(type);
        if (ctype == null) {
            System.err.println("Unsupported deploy type: " + type);
            System.exit(-1);
        }
        String srcAddr = null;
        if (ctype == Request.Deploy.Type.LM
                || ctype == Request.Deploy.Type.FWD
                || ctype == Request.Deploy.Type.FWDLM) {
            if (args.length < 5) {
                System.err.println("Src addr is required");
                System.exit(-1);
            }
            srcAddr = args[4];
        }
        int duration = PerformanceTest.DEFAULT_DURATION_SEC;
        for (int i = 1; i < args.length; ++i) {
            if ("-d".equals(args[i])) {
                if (i + 1 < args.length) {
                    duration = Integer.parseInt(args[i + 1]);
                } else {
                    System.err.println("-d requires duration(sec.)");
                    System.exit(-1);
                }
            } else if (args[i].startsWith("-")) {
                System.err.println("Ignored option: " + args[i]);
            }
        }
        new PerformanceTest().throughputTest(hostAddr, ctype, srcAddr, duration);
    }

    private static String deployTypeDescription() {
        return "[" + String.join("|", Arrays.stream(Request.Deploy.Type.values())
                                    .map(t -> t.name().toLowerCase())
                                    .collect(Collectors.toList())) + "]";
    }
}

package com.github.k4e;

import java.io.IOException;
import java.util.Arrays;
import java.util.UUID;
import java.util.stream.Collectors;

import com.github.k4e.exp.DeployTest;
import com.github.k4e.exp.SpeedTest;
import com.github.k4e.types.Request;

public class App {

    public static final UUID SESH_UUID = UUID.fromString("55C497AC-8AD5-4DA1-8673-6199443AE137");

    public static void main(String[] args) throws Exception {
        System.out.println("Build 2020-10-22");
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
            throw new UnsupportedOperationException(meth);
        }
    }

    private static void deploy(String[] args) throws IOException {
        if (args.length < 3) {
            System.err.printf("Required args: deploy %s <Host>\n", deployTypeDescription());
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
        if (ctype == Request.Deploy.Type.LM || ctype == Request.Deploy.Type.FWD) {
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
            System.err.println("Required args: remove <Host>");
            System.exit(-1);
        }
        String host = args[1];
        CloudletClient.remove(host);
    }

    private static void send(String[] args) throws IOException {
        if (args.length < 3) {
            System.err.println("Required args: send <Host> <Port> [message]");
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
            System.err.println("Required args: session <Host> <Port>");
            System.exit(-1);
        }
        String host = args[1];
        int port = Integer.parseInt(args[2]);
        // String fwdHost = null;
        // boolean resume = false;
        // for (int i = 3; i < args.length; ++i) {
        //     String arg = args[i];
        //     if ("-f".equals(arg) || "--forward".equals(arg)) {
        //         if (i + 1 < args.length) {
        //             fwdHost = args[i+1];
        //             ++i;
        //         } else {
        //             System.err.println("--forward requires hostIP");
        //             System.exit(-1);
        //         }
        //     } else if ("-r".equals(arg) || "--resume".equals(arg)) {
        //         resume = true;
        //     } else {
        //         System.err.println("Warning: ignored arg: " + arg);
        //     }
        // }
        // String fwdHostIp = null;
        // short fwdHostPort = 0;
        // if (fwdHost != null) {
        //     try {
        //         String[] ipPort = fwdHost.split(":");
        //         if (ipPort.length != 2) {
        //             throw new IllegalArgumentException();
        //         }
        //         fwdHostIp = ipPort[0];
        //         fwdHostPort = Short.parseShort(ipPort[1]);
        //     } catch (IllegalArgumentException e) {
        //         System.err.println("hostIp is expected as ipaddr:port but was %s" + fwdHost);
        //         System.exit(-1);
        //     }
        // }
        SessionClient.of(host, port, SESH_UUID).exec();
    }

    public static void experiment(String[] args) throws Exception {
        if (args.length < 2) {
            System.err.println("Experiment case required");
            System.exit(-1);
        }
        String expCase = args[1];
        switch (expCase) {
        case "deploy":
            deployTest(args);
            break;
        case "speed":
            speedTest(args);
            break;
        default:
            throw new UnsupportedOperationException(expCase);
        }
    }

    public static void deployTest(String[] args) throws Exception {
        if (args.length < 4) {
            System.err.printf(
                "Required args: experiment deploy <Host> %s\n", deployTypeDescription());
            System.exit(-1);
        }
        String host = args[2];
        String type = args[3];
        Request.Deploy.Type ctype = Request.Deploy.Type.valueOfIgnoreCase(type);
        if (ctype == null) {
            System.err.printf("Unsupported deploy type: " + type);
            System.exit(-1);
        }
        String srcAddr = null;
        if (ctype == Request.Deploy.Type.LM || ctype == Request.Deploy.Type.FWD) {
            if (args.length < 5) {
                System.err.println("Src-addr is required");
                System.exit(-1);
            }
            srcAddr = args[4];
        }
        new DeployTest(SESH_UUID).exec(host, ctype, srcAddr);
    }

    public static void speedTest(String[] args) throws Exception {
        if (args.length < 4) {
            System.err.println("Required args: experiment speed <Host> <DataSize(KB)>");
            System.exit(-1);
        }
        String hostAddr = args[2];
        int dataSize = Integer.parseInt(args[3]);
        int count = SpeedTest.DEFAULT_COUNT;
        Request.Deploy.Type ctype = null;
        String srcAddr = null;
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
            } else if ("-c".equals(args[i])) {
                if (i + 1 < args.length) {
                    String c = args[i+1];
                    count = Integer.parseInt(c);
                    ++i;
                } else {
                    System.err.println("-c requires count");
                    System.exit(-1);
                }
            } else if ("--deploy".equals(args[i])) {
                if (i + 1 < args.length) {
                    String type = args[i+1];
                    ctype = Request.Deploy.Type.valueOfIgnoreCase(type);
                    ++i;
                } else {
                    System.err.println("--deploy requires " + deployTypeDescription());
                    System.exit(-1);
                }
            } else if ("--src-addr".equals(args[i])) {
                if (i + 1 < args.length) {
                    srcAddr = args[i+1];
                    ++i;
                } else {
                    System.err.println("--src-addr requires source host address");
                    System.exit(-1);
                }
            }
        }
        if (ctype == Request.Deploy.Type.LM || ctype == Request.Deploy.Type.FWD) {
            if (srcAddr == null) {
                System.err.println("--src-addr is required");
                System.exit(-1);
            }
        }
        new SpeedTest(SESH_UUID).exec(hostAddr, ctype, dataSize, count, srcAddr, noWait, fullCheck);
    }

    private static String deployTypeDescription() {
        return "[" + String.join("|", Arrays.stream(Request.Deploy.Type.values())
                                    .map(t -> t.name().toLowerCase())
                                    .collect(Collectors.toList())) + "]";
    }
}

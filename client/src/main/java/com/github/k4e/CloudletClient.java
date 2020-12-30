package com.github.k4e;

import java.io.IOException;
import java.util.Map;

import com.github.k4e.types.Request;
import com.google.gson.Gson;

public class CloudletClient {

    public static String DEFAULT_NAME = "app-sample";
    public static String DEFAULT_IMAGE = "192.168.2.12:5000/app-sample";
    public static int DEFAULT_CLOUDLET_PORT = 9999;
    public static int DEFAULT_APP_IN_PORT = 8888;
    public static int DEFAULT_APP_EXT_PORT = 30088;

    public static Request createAppSampleRequest(
        Request.Deploy.Type type,
        String srcAddr,
        String dstAddr,
        Map<String, String> env,
        int bwLimit
    ) {
        Request.Deploy.Port appPort = new Request.Deploy.Port(DEFAULT_APP_IN_PORT, DEFAULT_APP_EXT_PORT);
        Request.Deploy.Port fwdPort = new Request.Deploy.Port(DEFAULT_APP_EXT_PORT, DEFAULT_APP_EXT_PORT); 
        Request.Deploy.NewApp newApp = null;
        Request.Deploy.Fwd fwd = null;
        Request.Deploy.LM lm = null;
        Request.Deploy.FwdLM fwdlm = null;
        if (type == Request.Deploy.Type.NEW) {
            newApp = new Request.Deploy.NewApp(DEFAULT_IMAGE, appPort, env);
        }
        if (type == Request.Deploy.Type.FWD) {
            fwd = new Request.Deploy.Fwd(srcAddr, fwdPort);
        }
        if (type == Request.Deploy.Type.LM) {
           lm = new Request.Deploy.LM(DEFAULT_IMAGE, srcAddr, DEFAULT_NAME, appPort, dstAddr, bwLimit);
        }
        if (type == Request.Deploy.Type.FWDLM) {
            fwdlm = new Request.Deploy.FwdLM(DEFAULT_IMAGE, srcAddr, DEFAULT_NAME, DEFAULT_APP_EXT_PORT, appPort, dstAddr, bwLimit);
        }
        return Request.deploy(new Request.Deploy(DEFAULT_NAME, type, newApp, fwd, lm, fwdlm));
    }

    public static void deploy(
        String host,
        Request.Deploy.Type type,
        String srcAddr,
        String dstAddr,
        Map<String, String> env,
        int bwLimit)
    throws IOException {
        Request req = createAppSampleRequest(type, srcAddr, dstAddr, env, bwLimit);
        send(host, DEFAULT_CLOUDLET_PORT, req);
    }

    public static void remove(String host) throws IOException {
        Request req = Request.remove(
            new Request.Remove(DEFAULT_NAME)
        );
        send(host, DEFAULT_CLOUDLET_PORT, req);
    }

    private static void send(String host, int port, Request req) throws IOException {
        Gson gson = new Gson();
        String msg = gson.toJson(req);
        send(host, port, msg);
    }

    private static void send(String host, int port, String msg) throws IOException {
        new SocketClient(host, port, msg).exec();
    }
}

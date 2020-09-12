package com.github.k4e;

import java.io.IOException;

import com.github.k4e.types.Request;
import com.google.common.collect.ImmutableMap;
import com.google.gson.Gson;

public class CloudletClient {

    public static void create(String host, int port, boolean createApp) throws IOException {
        Request req = Request.create(
            new Request.Create("app-sample", createApp, "k4edev/app-sample:latest", 8888, 30088,
                ImmutableMap.of("SLEEP_MS", "0"))
        );
        send(host, port, req);
    }

    public static void delete(String host, int port) throws IOException {
        Request req = Request.delete(
            new Request.Delete("app-sample")
        );
        send(host, port, req);
    }

    private static void send(String host, int port, Request req) throws IOException {
        Gson gson = new Gson();
        String msg = gson.toJson(req);
        send(host, port, msg);
    }

    private static void send(String host, int port, String msg) throws IOException {
        SocketSendRecv.sendRecv(host, port, msg);
    }
}

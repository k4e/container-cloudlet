package com.github.k4e;

import java.io.IOException;
import java.net.ServerSocket;
import java.net.Socket;
import java.net.SocketException;
import java.net.InetAddress;
import java.net.NetworkInterface;
import java.util.Collections;
import java.util.Enumeration;

import com.github.k4e.handler.Handler;
import com.github.k4e.handler.HandlerFactory;

public class Server {
    
    private final int port;

    public Server(int port) {
        this.port = port;
    }

    public void start(HandlerFactory handlerFactory) {
        ServerSocket sv = null;
        try {
            displayAddress();
            sv = new ServerSocket(port);
            while (true) {
                try {
                    Socket sock = sv.accept();
                    Handler handler = handlerFactory.create(sock);
                    Thread th = new Thread(handler);
                    th.start();
                } catch (IOException e) {
                    e.printStackTrace();
                }
            }
        } catch (IOException e) {
            e.printStackTrace();
        } finally {
            try {
                if (sv != null) {
                    sv.close();
                }
            } catch (IOException e) {
                e.printStackTrace();
            }
        }
    }

    public void displayAddress() throws SocketException {
        Enumeration<NetworkInterface> ifaces = NetworkInterface.getNetworkInterfaces();
        if (ifaces != null) {
            for (NetworkInterface iface : Collections.list(ifaces)) {
                System.out.printf("%s\t", iface.getName());
                for (InetAddress addr : Collections.list(iface.getInetAddresses())) {
                    System.out.printf("%s ", addr);
                }
                System.out.println();
            }
        } else {
            System.out.println("No interfaces");
        }
    }
}

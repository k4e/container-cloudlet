package com.github.k4e;

import java.io.IOException;
import java.net.Socket;

public interface SocketProc {

    void accept(Socket sock) throws IOException;
}

package com.github.k4e.types;

import java.net.Inet4Address;
import java.net.UnknownHostException;
import java.nio.ByteBuffer;
import java.util.UUID;

import com.google.common.base.Strings;
import com.google.common.primitives.Bytes;

public class ProtocolHeader {

    public static final int HEADER_SIZE = 21;
    public static final byte FLAG_RESUME = 0x1;

    public static ProtocolHeader create(UUID sessionId, String hostIP, boolean resume)
    throws UnknownHostException {
        if (Strings.isNullOrEmpty(hostIP)) {
            hostIP = "0.0.0.0";
        }
        byte[] bClientId = uuidToBytes(sessionId);
        byte[] bHostIP = ipv4ToBytes(hostIP);
        byte[] bFlag = createFlag(resume);
        return new ProtocolHeader(bClientId, bHostIP, bFlag);
    }

    public static byte[] uuidToBytes(UUID uuid) {
        ByteBuffer bb = ByteBuffer.wrap(new byte[16]);
        bb.putLong(uuid.getMostSignificantBits());
        bb.putLong(uuid.getLeastSignificantBits());
        byte[] ans = bb.array();
        int len = ans.length;
        if (len != 16) {
            throw new IllegalArgumentException(
                String.format("Invalid UUID length: %s (%d)", uuid, len)
            );
        }
        return ans;
    }

    public static byte[] ipv4ToBytes(String ipv4) throws UnknownHostException {
        return Inet4Address.getByName(ipv4).getAddress();
    }

    public static byte[] createFlag(boolean resume) {
        byte[] ans = new byte[] { 0x0 };
        if (resume) {
            ans[0] |= FLAG_RESUME;
        }
        return ans;
    }

    private final byte[] sessionId;
    private final byte[] hostIP;
    private final byte[] flag;

    public ProtocolHeader(byte[] sessionId, byte[] hostIP, byte[] flag) {
        this.sessionId = sessionId;
        this.hostIP = hostIP;
        this.flag = flag;
    }

    public byte[] getBytes() throws IllegalStateException {
        byte[] ans = Bytes.concat(sessionId, hostIP, flag);
        int len = ans.length;
        if (len != HEADER_SIZE) {
            throw new IllegalStateException(String.format("Unexpected header size: %d", len));
        }
        return ans;
    }

    @Override
    public String toString() {
        ByteBuffer uuidBB = ByteBuffer.wrap(sessionId);
        Long uuidHigh = uuidBB.getLong();
        Long uuidLow = uuidBB.getLong();
        String sSessionId = new UUID(uuidHigh, uuidLow).toString();
        String sHostIP = String.format("%d.%d.%d.%d", hostIP[0], hostIP[1], hostIP[2], hostIP[3]);
        String sFlag = String.format("%02x", flag[0]);
        return String.format("sessionId=%s, hostIP=%s, flag=%s", sSessionId, sHostIP, sFlag);
    }
}

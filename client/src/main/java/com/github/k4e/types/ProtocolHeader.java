package com.github.k4e.types;

// import java.net.Inet4Address;
import java.net.UnknownHostException;
import java.nio.ByteBuffer;
// import java.nio.ByteOrder;
import java.util.UUID;

// import com.google.common.base.Strings;
import com.google.common.primitives.Bytes;

public class ProtocolHeader {

    public static final int HEADER_SIZE = 16;
    public static final byte FLAG_RESUME = 0x1;

    public static ProtocolHeader create(UUID sessionId/*, String dstIP, short dstPort, boolean resume*/)
    throws UnknownHostException {
        // if (Strings.isNullOrEmpty(dstIP)) {
        //     dstIP = "0.0.0.0";
        // }
        byte[] bSessionId = uuidToBytes(sessionId);
        // byte[] bHostIP = ipv4ToBytes(dstIP);
        // byte[] bPort = shortToBytesBigEndian(dstPort);
        // byte[] bFlag = createFlag(resume);
        return new ProtocolHeader(bSessionId/*, bHostIP, bPort, bFlag*/);
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

    // public static byte[] ipv4ToBytes(String ipv4) throws UnknownHostException {
    //     return Inet4Address.getByName(ipv4).getAddress();
    // }

    // public static byte[] shortToBytesBigEndian(short h) {
    //     byte b0 = (byte)((h >> 8) & 0xff);
    //     byte b1 = (byte)(h & 0xff);
    //     return new byte[] { b0 , b1 };
    // }

    // public static byte[] createFlag(boolean resume) {
    //     byte[] ans = new byte[] { 0x0 };
    //     if (resume) {
    //         ans[0] |= FLAG_RESUME;
    //     }
    //     return ans;
    // }

    private final byte[] sessionId;
    // private final byte[] dstIP;
    // private final byte[] dstPort;
    // private final byte[] flag;

    public ProtocolHeader(byte[] sessionId/*, byte[] dstIP, byte[] dstPort, byte[] flag*/) {
        this.sessionId = sessionId;
        // this.dstIP = dstIP;
        // this.dstPort = dstPort;
        // this.flag = flag;
    }

    public byte[] getBytes() throws IllegalStateException {
        byte[] ans = Bytes.concat(sessionId/*, dstIP, dstPort, flag*/);
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
        // String sIP = String.format("%d.%d.%d.%d",
        //         dstIP[0] & 0xff, dstIP[1] & 0xff, dstIP[2] & 0xff, dstIP[3] & 0xff);
        // String sPort = Short.toString(ByteBuffer.wrap(dstPort).order(ByteOrder.BIG_ENDIAN).getShort());
        // String sFlag = String.format("%02x", flag[0]);
        // return String.format("sessionId=%s, hostIP=%s, port=%s, flag=%s", sSessionId, sIP, sPort, sFlag);
        return String.format("sessionId=%s", sSessionId);
    }
}

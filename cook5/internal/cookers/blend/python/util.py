import codecs
import struct


UTF8Writer = codecs.getwriter('utf-8')


def float32_bits(x):
    return struct.unpack('<I', struct.pack('<f', x))[0]

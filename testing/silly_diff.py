#!/usr/bin/env python

import binascii
import sys

a = binascii.unhexlify(sys.argv[1])
b = binascii.unhexlify(sys.argv[2])

a = a.replace('X', 'Y')

if b == a:
  sys.exit(0)
else:
  print "A != B:"
  print sys.argv[1]
  print a
  print sys.argv[2]
  print b
  sys.exit(100)

CC=gcc
NETSNMP_CONFIG=net-snmp-config
CFLAGS=$(shell $(NETSNMP_CONFIG) --cflags) -fPIC -Wall -Wextra
LDFLAGS=$(shell $(NETSNMP_CONFIG) --libs) -shared

TARGET=sssonector.so
SOURCES=sssonector.c
OBJECTS=$(SOURCES:.c=.o)

all: $(TARGET)

$(TARGET): $(OBJECTS)
$(CC) -o $@ $^ $(LDFLAGS)

%.o: %.c
$(CC) $(CFLAGS) -c $< -o $@

install: $(TARGET)
@mkdir -p /usr/lib/snmp/dlmod
@cp $(TARGET) /usr/lib/snmp/dlmod/
@echo "dlmod sssonector /usr/lib/snmp/dlmod/sssonector.so" > /etc/snmp/snmpd.conf.d/sssonector.conf
@systemctl restart snmpd

clean:
rm -f $(OBJECTS) $(TARGET)

.PHONY: all install clean

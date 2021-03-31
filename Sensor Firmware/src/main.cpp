#include <Arduino.h>
#include <WiFiNINA.h>
#include <Servo.h>

#include "WiFi Private.h"

#define BLTOUCH_TX 10
#define BLTOUCH_RX 5

WiFiServer server(9988);
Servo BLTouch;

volatile bool touched = false;

void onTouched() {
    touched = true;
    BLTouch.write(90);
}

void setup() {
    char ssid[] = WIFI_SSID;
    char pass[] = WIFI_PASSWORD;

    Serial.begin(9600);

    int wifiStatus = WL_IDLE_STATUS;

    if (WiFi.status() == WL_NO_MODULE) {
        Serial.println("Communication with WiFi module failed!");
        while (true);
    }

    String fv = WiFi.firmwareVersion();
    if (fv < WIFI_FIRMWARE_LATEST_VERSION) {
        Serial.println("Please upgrade the firmware");
    }

    while (wifiStatus != WL_CONNECTED) {
        Serial.print("Attempting to connect to WPA SSID: ");
        Serial.println(ssid);
        wifiStatus = WiFi.begin(ssid, pass);
        // Wait 10 seconds for connection
        delay(10000);
    }

    Serial.println("Connected!");
    Serial.print("IP Address: ");
    Serial.println(WiFi.localIP());
    Serial.print("RSSI: ");
    Serial.println(WiFi.RSSI());

    server.begin();

    pinMode(BLTOUCH_TX, OUTPUT);
    pinMode(BLTOUCH_RX, INPUT_PULLUP);
    BLTouch.attach(BLTOUCH_TX);
    BLTouch.write(90);
    attachInterrupt(BLTOUCH_RX, onTouched, RISING);


    Serial.println("Listening");
}

void handleCommand(WiFiClient& client, char command) {
    Serial.println("Got command: " + command);
    if (command == 'e') {
        BLTouch.write(10);
    } else if (command == 'r') {
        BLTouch.write(90);
    } else if (command == 't') {
        client.write(touched ? '1' : '0');
        touched = false;
    }
}

void loop() {
    auto client = server.available();
    if (client && client.available()) {
        int command = client.read();
        if (command > 0) {
            handleCommand(client, static_cast<char>(command));
        }
    }
}
#include <Adafruit_NeoPixel.h>
#ifdef __AVR__
  #include <avr/power.h>
#endif

#define PIN A0
#define NUMPIXELS 8

Adafruit_NeoPixel pixels = Adafruit_NeoPixel(NUMPIXELS, PIN, NEO_GRB + NEO_KHZ800);

void setup() {
  Serial.begin(9600);
  Serial.setTimeout(60000);

  pixels.begin();

  for(int i=0;i<NUMPIXELS;i++){
    pixels.setPixelColor(i, pixels.Color(0,0,0));
  }

  pixels.show();
}

void loop() {
  outer: while (1) {
    auto line = Serial.readStringUntil(10);

    if (0 == line.length()) {
      continue;
    }

    while (line.endsWith("\r") || line.endsWith("\n") && line.length()) {
      int lastIndex = line.length() - 1;

      line.remove(lastIndex, 1);
    }

    if (line.equals("AT")) {
      Serial.println("OK");
    } else if ((line.startsWith("AT+RGB=")) && (13 == line.length())) {
      const char* args = line.c_str() + 7;

      for (int i = 0; i < 6; i++) {
        char ch = args[i];

        if (! isHexadecimalDigit(ch)) {
          Serial.println("ERROR: INVALID VALUE");
          goto outer;
        }
      }

      const char* b = args + 4;
      const char* g = args + 2;
      const char* r = args;

      char buf[3];

      uint8_t rB, gB, bB;

      strncpy(buf, r, 2);
      rB = (uint8_t) strtol(buf, NULL, 16);

      strncpy(buf, g, 2);
      gB = (uint8_t) strtol(buf, NULL, 16);

      strncpy(buf, b, 2);
      bB = (uint8_t) strtol(buf, NULL, 16);

      pixels.setBrightness(0xFF);

      for(int i=0;i<NUMPIXELS;i++){
        pixels.setPixelColor(i, pixels.Color(rB,gB,bB));
        pixels.show(); // This sends the updated pixel color to the hardware.

        delay(200);
      }

      Serial.println("OK");
    } else if (line.equals("AT+OFF")) {
      for(int i=0;i<NUMPIXELS;i++){
        pixels.setPixelColor(i, pixels.Color(0,0,0));
      }

      pixels.show();

      Serial.println("OK");
    } else if ((line.startsWith("AT+BRIGHTNESS=") && (16 == line.length()))) {
      char buf[3];

      strncpy(buf, line.c_str() + 14, 2);
      
      uint8_t brightness = (uint8_t) strtol(buf, NULL, 16);

      pixels.setBrightness(brightness);

      pixels.show();

      Serial.println("OK");
    } else {
      Serial.println("ERROR");
    }
  }
}

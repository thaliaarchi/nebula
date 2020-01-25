#include <stdio.h>
#include <stdint.h>

void printc(int64_t c) {
  fputc(c, stdout);
}

void printi(int64_t i) {
  printf("%d", (int) i);
}

int64_t readc() {
  return fgetc(stdout);
}

int64_t readi() {
  int i;
  scanf("%d", &i);
  return i;
}

void flush() {
  fflush(stdout);
}

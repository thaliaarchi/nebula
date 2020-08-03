#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

extern uint64_t stack_len;
extern uint64_t call_stack_len;

void printc(int64_t c) {
  fputc(c, stdout);
}

void printi(int64_t i) {
  printf("%d", (int) i);
}

int64_t readc() {
  return fgetc(stdin);
}

int64_t readi() {
  int i;
  fscanf(stdin, "%d", &i);
  return i;
}

void flush() {
  fflush(stdout);
}

// TODO change to procedure generated in IR to enable transformations.
void check_stack(uint64_t n, char *block, char *pos) {
  if (stack_len < n) {
    fprintf(stderr, "Data stack underflow in %s at %s\n", block, pos);
    fflush(stderr);
    exit(1);
  }
}

// TODO change to procedure generated in IR to enable transformations.
void check_call_stack(char *block, char *pos) {
  if (call_stack_len < 1) {
    fprintf(stderr, "Call stack underflow in %s at %s\n", block, pos);
    fflush(stderr);
    exit(1);
  }
}

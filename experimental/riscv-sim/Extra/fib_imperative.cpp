


int fib_imperative( int n ) {
    int fib1 = 1;
    int fib2 = 1;
    int fibonacci = fib1;
    int i = 2;
    while ( i < n ) {
        fibonacci = fib1 + fib2;
        fib1 = fib2;
        fib2 = fibonacci;
        i = i + 1;
    }
    return fibonacci;
}


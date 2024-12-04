
int fib_recursive( int n ) {
    if ( n < 2 ) {
        return n;
    }
    return fib_recursive( n - 1 ) + fib_recursive( n - 2 );
}

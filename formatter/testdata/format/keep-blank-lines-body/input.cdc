access(all) contract Test {
    init() {
        self.a = 1
        self.b = 2

        self.c = 3

        self.d = 4
        self.e = 5
    }

    access(all) fun example() {
        let x = 1

        let y = 2
        let z = 3
    }
}

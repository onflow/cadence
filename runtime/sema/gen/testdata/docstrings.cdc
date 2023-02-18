pub struct Docstrings {
    /// This is a 1-line docstring.
    pub let owo: Int

    /// This is a 2-line docstring.
    /// This is the second line.
    pub let uwu: [Int]

    /// This is a 3-line docstring for a function.
    /// This is the second line.
    /// And the third line!
    pub fun nwn(x: Int): String?

    /// This is a multiline docstring.
    ///
    /// There should be two newlines before this line!
    pub let withBlanks: Int

    /// The function `isSmolBean` has docstrings with backticks.
    /// These should be handled accordingly.
    pub fun isSmolBean(): Bool

    /// A function with a docstring.
    /// This docstring is `cool` because it has inline backticked expressions.
    /// Look, I did it `again`, wowie!!
    pub fun runningOutOfIdeas(): UInt64?

}
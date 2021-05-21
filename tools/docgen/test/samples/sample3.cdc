///
///      This is a function, with too many spaces in docs.
///
///
///@param name: The name. Must be a string
///
///
///
///        @param bytes: Content to be validated
///
///         @return Validity of the content
///
///
fun func_1(name: String, bytes: [Int8]): bool {
}


///
/// This function doc contains mixed params and doc lines.
///
/// @param name: The name. Must be a string
///
/// Some doc line in the middle of parameters.
///
/// @param bytes: Content to be validated
///
/// @return Validity of the content
///
/// Another doc line at the end.
///
fun func_2(name: String, bytes: [Int8]): bool {
}

///
/// param1's name is missing.
/// @param : The param1. Must be a string
///
/// param2's colon is missing.
/// @param param2 Content to be validated
///
/// param3's description is missing
/// @param param3 :
///
/// param4's description is missing including colon
/// @param param4
///
/// param5's everything is missing
/// @param
///
fun func_3() {
}

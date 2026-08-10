/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package bbq

// Export is a construct of a program that other programs may import.
//
// It is the counterpart of Import: while Import records what a program needs
// from elsewhere, Export records what a program offers to others.
// Keeping this explicit means importers do not have to infer a program's
// public interface from its internal globals, which also contain constructs
// that are not importable, such as members nested inside a type.
//
// Aliases are deliberately absent: an alias (e.g. `import Foo as Bar`)
// is a property of the importing program, not of the exporting one.
//
// NOTE: membership is currently determined by whether a construct is declared
// at the top level, not by its access modifier, so a non-public top-level
// construct is still listed. The interpreter is equally permissive: a wildcard
// import copies every global of the imported program. Narrowing this to public
// constructs only would have to happen in both places at once.
type Export struct {
	CanonicalName CanonicalName
}

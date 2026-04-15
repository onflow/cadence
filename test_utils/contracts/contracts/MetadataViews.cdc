import "FungibleToken"
import "NonFungibleToken"
import "ViewResolver"

/// This contract implements the metadata standard proposed
/// in FLIP-0636.
///
/// Ref: https://github.com/onflow/flips/blob/main/application/20210916-nft-metadata.md
///
/// Structs and resources can implement one or more
/// metadata types, called views. Each view type represents
/// a different kind of metadata, such as a creator biography
/// or a JPEG image file.
///
access(all) contract MetadataViews {

    /// Display is a basic view that includes the name, description and
    /// thumbnail for an object. Most objects should implement this view.
    ///
    access(all) struct Display {

        /// The name of the object.
        ///
        /// This field will be displayed in lists and therefore should
        /// be short an concise.
        ///
        access(all) let name: String

        /// A written description of the object.
        ///
        /// This field will be displayed in a detailed view of the object,
        /// so can be more verbose (e.g. a paragraph instead of a single line).
        ///
        access(all) let description: String

        /// A small thumbnail representation of the object.
        ///
        /// This field should be a web-friendly file (i.e JPEG, PNG)
        /// that can be displayed in lists, link previews, etc.
        ///
        access(all) let thumbnail: {File}

        view init(
            name: String,
            description: String,
            thumbnail: {File}
        ) {
            self.name = name
            self.description = description
            self.thumbnail = thumbnail
        }
    }

    /// Helper to get Display in a typesafe way
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return An optional Display struct
    ///
    access(all) fun getDisplay(_ viewResolver: &{ViewResolver.Resolver}) : Display? {
        if let view = viewResolver.resolveView(Type<Display>()) {
            if let v = view as? Display {
                return v
            }
        }
        return nil
    }

    /// Generic interface that represents a file stored on or off chain. Files
    /// can be used to references images, videos and other media.
    ///
    access(all) struct interface File {
        access(all) view fun uri(): String
    }

    /// View to expose a file that is accessible at an HTTP (or HTTPS) URL.
    ///
    access(all) struct HTTPFile: File {
        access(all) let url: String

        view init(url: String) {
            self.url = url
        }

        access(all) view fun uri(): String {
            return self.url
        }
    }

    /// View to expose a file stored on IPFS.
    /// IPFS images are referenced by their content identifier (CID)
    /// rather than a direct URI. A client application can use this CID
    /// to find and load the image via an IPFS gateway.
    ///
    access(all) struct IPFSFile: File {

        /// CID is the content identifier for this IPFS file.
        ///
        /// Ref: https://docs.ipfs.io/concepts/content-addressing/
        ///
        access(all) let cid: String

        /// Path is an optional path to the file resource in an IPFS directory.
        ///
        /// This field is only needed if the file is inside a directory.
        ///
        /// Ref: https://docs.ipfs.io/concepts/file-systems/
        ///
        access(all) let path: String?

        view init(cid: String, path: String?) {
            self.cid = cid
            self.path = path
        }

        /// This function returns the IPFS native URL for this file.
        /// Ref: https://docs.ipfs.io/how-to/address-ipfs-on-web/#native-urls
        ///
        /// @return The string containing the file uri
        ///
        access(all) view fun uri(): String {
            if let path = self.path {
                return "ipfs://".concat(self.cid).concat("/").concat(path)
            }

            return "ipfs://".concat(self.cid)
        }
    }

    /// A struct to represent a generic URI. May be used to represent the URI of
    /// the NFT where the type of URI is not able to be determined (i.e. HTTP,
    /// IPFS, etc.)
    ///
    access(all) struct URI: File {
        /// The base URI prefix, if any. Not needed for all URIs, but helpful
        /// for some use cases For example, updating a whole NFT collection's
        /// image host easily
        ///
        access(all) let baseURI: String?
        /// The URI string value
        /// NOTE: this is set on init as a concatenation of the baseURI and the
        /// value if baseURI != nil
        ///
        access(self) let value: String

        access(all) view fun uri(): String {
            return self.value
        }

        init(baseURI: String?, value: String) {
            self.baseURI = baseURI
            self.value = baseURI != nil ? baseURI!.concat(value) : value
        }
    }

    access(all) struct Media {

        /// File for the media
        ///
        access(all) let file: {File}

        /// media-type comes on the form of type/subtype as described here
        /// https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types
        ///
        access(all) let mediaType: String

        view init(file: {File}, mediaType: String) {
          self.file=file
          self.mediaType=mediaType
        }
    }

    /// Wrapper view for multiple media views
    ///
    access(all) struct Medias {

        /// An arbitrary-sized list for any number of Media items
        access(all) let items: [Media]

        view init(_ items: [Media]) {
            self.items = items
        }
    }

    /// Helper to get Medias in a typesafe way
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return A optional Medias struct
    ///
    access(all) fun getMedias(_ viewResolver: &{ViewResolver.Resolver}) : Medias? {
        if let view = viewResolver.resolveView(Type<Medias>()) {
            if let v = view as? Medias {
                return v
            }
        }
        return nil
    }

    /// View to represent a license according to https://spdx.org/licenses/
    /// This view can be used if the content of an NFT is licensed.
    ///
    access(all) struct License {
        access(all) let spdxIdentifier: String

        view init(_ identifier: String) {
            self.spdxIdentifier = identifier
        }
    }

    /// Helper to get License in a typesafe way
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return An optional License struct
    ///
    access(all) fun getLicense(_ viewResolver: &{ViewResolver.Resolver}) : License? {
        if let view = viewResolver.resolveView(Type<License>()) {
            if let v = view as? License {
                return v
            }
        }
        return nil
    }

    /// View to expose a URL to this item on an external site.
    /// This can be used by applications like .find and Blocto to direct users
    /// to the original link for an NFT or a project page that describes the NFT collection.
    /// eg https://www.my-nft-project.com/overview-of-nft-collection
    ///
    access(all) struct ExternalURL {
        access(all) let url: String

        view init(_ url: String) {
            self.url=url
        }
    }

    /// Helper to get ExternalURL in a typesafe way
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return An optional ExternalURL struct
    ///
    access(all) fun getExternalURL(_ viewResolver: &{ViewResolver.Resolver}) : ExternalURL? {
        if let view = viewResolver.resolveView(Type<ExternalURL>()) {
            if let v = view as? ExternalURL {
                return v
            }
        }
        return nil
    }

    /// View that defines the composable royalty standard that gives marketplaces a
    /// unified interface to support NFT royalties.
    ///
    access(all) struct Royalty {

        /// Generic FungibleToken Receiver for the beneficiary of the royalty
        /// Can get the concrete type of the receiver with receiver.getType()
        /// Recommendation - Users should create a new link for a FlowToken
        /// receiver for this using `getRoyaltyReceiverPublicPath()`, and not
        /// use the default FlowToken receiver. This will allow users to update
        /// the capability in the future to use a more generic capability
        access(all) let receiver: Capability<&{FungibleToken.Receiver}>

        /// Multiplier used to calculate the amount of sale value transferred to
        /// royalty receiver. Note - It should be between 0.0 and 1.0
        /// Ex - If the sale value is x and multiplier is 0.56 then the royalty
        /// value would be 0.56 * x.
        /// Generally percentage get represented in terms of basis points
        /// in solidity based smart contracts while cadence offers `UFix64`
        /// that already supports the basis points use case because its
        /// operations are entirely deterministic integer operations and support
        /// up to 8 points of precision.
        access(all) let cut: UFix64

        /// Optional description: This can be the cause of paying the royalty,
        /// the relationship between the `wallet` and the NFT, or anything else
        /// that the owner might want to specify.
        access(all) let description: String

        view init(receiver: Capability<&{FungibleToken.Receiver}>, cut: UFix64, description: String) {
            pre {
                cut >= 0.0 && cut <= 1.0 :
                    "MetadataViews.Royalty.init: Cannot initialize the Royalty Metadata View! "
                    .concat("The provided royalty cut value of ").concat(cut.toString())
                    .concat(" is invalid. ")
                    .concat("It should be within the valid range between 0 and 1. i.e [0,1]")
            }
            self.receiver = receiver
            self.cut = cut
            self.description = description
        }
    }

    /// Wrapper view for multiple Royalty views.
    /// Marketplaces can query this `Royalties` struct from NFTs
    /// and are expected to pay royalties based on these specifications.
    ///
    access(all) struct Royalties {

        /// Array that tracks the individual royalties
        access(self) let cutInfos: [Royalty]

        access(all) view init(_ cutInfos: [Royalty]) {
            // Validate that sum of all cut multipliers should not be greater than 1.0
            var totalCut = 0.0
            for royalty in cutInfos {
                totalCut = totalCut + royalty.cut
            }
            assert(
                totalCut <= 1.0,
                message:
                    "MetadataViews.Royalties.init: Cannot initialize Royalties Metadata View! "
                    .concat(" The sum of cutInfos multipliers is ")
                    .concat(totalCut.toString())
                    .concat(" but it should not be greater than 1.0")
            )
            // Assign the cutInfos
            self.cutInfos = cutInfos
        }

        /// Return the cutInfos list
        ///
        /// @return An array containing all the royalties structs
        ///
        access(all) view fun getRoyalties(): [Royalty] {
            return self.cutInfos
        }
    }

    /// Helper to get Royalties in a typesafe way
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return A optional Royalties struct
    ///
    access(all) fun getRoyalties(_ viewResolver: &{ViewResolver.Resolver}) : Royalties? {
        if let view = viewResolver.resolveView(Type<Royalties>()) {
            if let v = view as? Royalties {
                return v
            }
        }
        return nil
    }

    /// Get the path that should be used for receiving royalties
    /// This is a path that will eventually be used for a generic switchboard receiver,
    /// hence the name but will only be used for royalties for now.
    ///
    /// @return The PublicPath for the generic FT receiver
    ///
    access(all) view fun getRoyaltyReceiverPublicPath(): PublicPath {
        return /public/GenericFTReceiver
    }

    /// View to represent a single field of metadata on an NFT.
    /// This is used to get traits of individual key/value pairs along with some
    /// contextualized data about the trait
    ///
    access(all) struct Trait {
        // The name of the trait. Like Background, Eyes, Hair, etc.
        access(all) let name: String

        // The underlying value of the trait, the rest of the fields of a trait provide context to the value.
        access(all) let value: AnyStruct

        // displayType is used to show some context about what this name and value represent
        // for instance, you could set value to a unix timestamp, and specify displayType as "Date" to tell
        // platforms to consume this trait as a date and not a number
        access(all) let displayType: String?

        // Rarity can also be used directly on an attribute.
        //
        // This is optional because not all attributes need to contribute to the NFT's rarity.
        access(all) let rarity: Rarity?

        view init(name: String, value: AnyStruct, displayType: String?, rarity: Rarity?) {
            self.name = name
            self.value = value
            self.displayType = displayType
            self.rarity = rarity
        }
    }

    /// Wrapper view to return all the traits on an NFT.
    /// This is used to return traits as individual key/value pairs along with
    /// some contextualized data about each trait.
    access(all) struct Traits {
        access(all) let traits: [Trait]

        view init(_ traits: [Trait]) {
            self.traits = traits
        }

        /// Adds a single Trait to the Traits view
        ///
        /// @param Trait: The trait struct to be added
        ///
        access(all) fun addTrait(_ t: Trait) {
            self.traits.append(t)
        }
    }

    /// Helper to get Traits view in a typesafe way
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return A optional Traits struct
    ///
    access(all) fun getTraits(_ viewResolver: &{ViewResolver.Resolver}) : Traits? {
        if let view = viewResolver.resolveView(Type<Traits>()) {
            if let v = view as? Traits {
                return v
            }
        }
        return nil
    }

    /// Helper function to easily convert a dictionary to traits. For NFT
    /// collections that do not need either of the optional values of a Trait,
    /// this method should suffice to give them an array of valid traits.
    ///
    /// @param dict: The dictionary to be converted to Traits
    /// @param excludedNames: An optional String array specifying the `dict`
    ///         keys that are not wanted to become `Traits`
    /// @return The generated Traits view
    ///
    access(all) fun dictToTraits(dict: {String: AnyStruct}, excludedNames: [String]?): Traits {
        // Collection owners might not want all the fields in their metadata included.
        // They might want to handle some specially, or they might just not want them included at all.
        if excludedNames != nil {
            for k in excludedNames! {
                dict.remove(key: k)
            }
        }

        let traits: [Trait] = []
        for k in dict.keys {
            let trait = Trait(name: k, value: dict[k]!, displayType: nil, rarity: nil)
            traits.append(trait)
        }

        return Traits(traits)
    }

    /// Optional view for collections that issue multiple objects
    /// with the same or similar metadata, for example an X of 100 set. This
    /// information is useful for wallets and marketplaces.
    /// An NFT might be part of multiple editions, which is why the edition
    /// information is returned as an arbitrary sized array
    ///
    access(all) struct Edition {

        /// The name of the edition
        /// For example, this could be Set, Play, Series,
        /// or any other way a project could classify its editions
        access(all) let name: String?

        /// The edition number of the object.
        /// For an "24 of 100 (#24/100)" item, the number is 24.
        access(all) let number: UInt64

        /// The max edition number of this type of objects.
        /// This field should only be provided for limited-editioned objects.
        /// For an "24 of 100 (#24/100)" item, max is 100.
        /// For an item with unlimited edition, max should be set to nil.
        ///
        access(all) let max: UInt64?

        view init(name: String?, number: UInt64, max: UInt64?) {
            if max != nil {
                assert(
                    number <= max!,
                    message:
                        "MetadataViews.Edition.init: Cannot intialize the Edition Metadata View! "
                        .concat("The provided edition number of ")
                        .concat(number.toString())
                        .concat(" cannot be greater than the max edition number of ")
                        .concat(max!.toString())
                        .concat(".")
                )
            }
            self.name = name
            self.number = number
            self.max = max
        }
    }

    /// Wrapper view for multiple Edition views
    ///
    access(all) struct Editions {

        /// An arbitrary-sized list for any number of editions
        /// that the NFT might be a part of
        access(all) let infoList: [Edition]

        view init(_ infoList: [Edition]) {
            self.infoList = infoList
        }
    }

    /// Helper to get Editions in a typesafe way
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return An optional Editions struct
    ///
    access(all) fun getEditions(_ viewResolver: &{ViewResolver.Resolver}) : Editions? {
        if let view = viewResolver.resolveView(Type<Editions>()) {
            if let v = view as? Editions {
                return v
            }
        }
        return nil
    }

    /// View representing a project-defined serial number for a specific NFT
    /// Projects have different definitions for what a serial number should be
    /// Some may use the NFTs regular ID and some may use a different
    /// classification system. The serial number is expected to be unique among
    /// other NFTs within that project
    ///
    access(all) struct Serial {
        access(all) let number: UInt64

        view init(_ number: UInt64) {
            self.number = number
        }
    }

    /// Helper to get Serial in a typesafe way
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return An optional Serial struct
    ///
    access(all) fun getSerial(_ viewResolver: &{ViewResolver.Resolver}) : Serial? {
        if let view = viewResolver.resolveView(Type<Serial>()) {
            if let v = view as? Serial {
                return v
            }
        }
        return nil
    }

    /// View to expose rarity information for a single rarity
    /// Note that a rarity needs to have either score or description but it can
    /// have both
    ///
    access(all) struct Rarity {
        /// The score of the rarity as a number
        access(all) let score: UFix64?

        /// The maximum value of score
        access(all) let max: UFix64?

        /// The description of the rarity as a string.
        ///
        /// This could be Legendary, Epic, Rare, Uncommon, Common or any other string value
        access(all) let description: String?

        view init(score: UFix64?, max: UFix64?, description: String?) {
            if score == nil && description == nil {
                panic("MetadataViews.Rarity.init: Cannot initialize the Rarity Metadata View! "
                      .concat("The provided score and description are both `nil`. ")
                      .concat(" A Rarity needs to set score, description, or both"))
            }

            self.score = score
            self.max = max
            self.description = description
        }
    }

    /// Helper to get Rarity view in a typesafe way
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return A optional Rarity struct
    ///
    access(all) fun getRarity(_ viewResolver: &{ViewResolver.Resolver}) : Rarity? {
        if let view = viewResolver.resolveView(Type<Rarity>()) {
            if let v = view as? Rarity {
                return v
            }
        }
        return nil
    }

    /// NFTView wraps all Core views along `id` and `uuid` fields, and is used
    /// to give a complete picture of an NFT. Most NFTs should implement this
    /// view.
    ///
    access(all) struct NFTView {
        access(all) let id: UInt64
        access(all) let uuid: UInt64
        access(all) let display: MetadataViews.Display?
        access(all) let externalURL: MetadataViews.ExternalURL?
        access(all) let collectionData: NFTCollectionData?
        access(all) let collectionDisplay: NFTCollectionDisplay?
        access(all) let royalties: Royalties?
        access(all) let traits: Traits?

        view init(
            id : UInt64,
            uuid : UInt64,
            display : MetadataViews.Display?,
            externalURL : MetadataViews.ExternalURL?,
            collectionData : NFTCollectionData?,
            collectionDisplay : NFTCollectionDisplay?,
            royalties : Royalties?,
            traits: Traits?
        ) {
            self.id = id
            self.uuid = uuid
            self.display = display
            self.externalURL = externalURL
            self.collectionData = collectionData
            self.collectionDisplay = collectionDisplay
            self.royalties = royalties
            self.traits = traits
        }
    }

    /// Helper to get an NFT view
    ///
    /// @param id: The NFT id
    /// @param viewResolver: A reference to the resolver resource
    /// @return A NFTView struct
    ///
    access(all) fun getNFTView(id: UInt64, viewResolver: &{ViewResolver.Resolver}) : NFTView {
        let nftView = viewResolver.resolveView(Type<NFTView>())
        if nftView != nil {
            return nftView! as! NFTView
        }

        return NFTView(
            id : id,
            uuid: viewResolver.uuid,
            display: MetadataViews.getDisplay(viewResolver),
            externalURL : MetadataViews.getExternalURL(viewResolver),
            collectionData : self.getNFTCollectionData(viewResolver),
            collectionDisplay : self.getNFTCollectionDisplay(viewResolver),
            royalties : self.getRoyalties(viewResolver),
            traits : self.getTraits(viewResolver)
        )
    }

    /// View to expose the information needed store and retrieve an NFT.
    /// This can be used by applications to setup a NFT collection with proper
    /// storage and public capabilities.
    ///
    access(all) struct NFTCollectionData {
        /// Path in storage where this NFT is recommended to be stored.
        access(all) let storagePath: StoragePath

        /// Public path which must be linked to expose public capabilities of this NFT
        /// including standard NFT interfaces and metadataviews interfaces
        access(all) let publicPath: PublicPath

        /// The concrete type of the collection that is exposed to the public
        /// now that entitlements exist, it no longer needs to be restricted to a specific interface
        access(all) let publicCollection: Type

        /// Type that should be linked at the aforementioned public path
        access(all) let publicLinkedType: Type

        /// Function that allows creation of an empty NFT collection that is intended to store
        /// this NFT.
        access(all) let createEmptyCollection: fun(): @{NonFungibleToken.Collection}

        view init(
            storagePath: StoragePath,
            publicPath: PublicPath,
            publicCollection: Type,
            publicLinkedType: Type,
            createEmptyCollectionFunction: fun(): @{NonFungibleToken.Collection}
        ) {
            pre {
                publicLinkedType.isSubtype(of: Type<&{NonFungibleToken.Collection}>()):
                    "MetadataViews.NFTCollectionData.init: Cannot initialize the NFTCollectionData Metadata View! "
                    .concat("The Public linked type <")
                    .concat(publicLinkedType.identifier)
                    .concat("> is incorrect. It must be a subtype of the NonFungibleToken.Collection interface.")
            }
            self.storagePath=storagePath
            self.publicPath=publicPath
            self.publicCollection=publicCollection
            self.publicLinkedType=publicLinkedType
            self.createEmptyCollection=createEmptyCollectionFunction
        }
    }

    /// Helper to get NFTCollectionData in a way that will return an typed Optional
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return A optional NFTCollectionData struct
    ///
    access(all) fun getNFTCollectionData(_ viewResolver: &{ViewResolver.Resolver}) : NFTCollectionData? {
        if let view = viewResolver.resolveView(Type<NFTCollectionData>()) {
            if let v = view as? NFTCollectionData {
                return v
            }
        }
        return nil
    }

    /// View to expose the information needed to showcase this NFT's
    /// collection. This can be used by applications to give an overview and
    /// graphics of the NFT collection this NFT belongs to.
    ///
    access(all) struct NFTCollectionDisplay {
        // Name that should be used when displaying this NFT collection.
        access(all) let name: String

        // Description that should be used to give an overview of this collection.
        access(all) let description: String

        // External link to a URL to view more information about this collection.
        access(all) let externalURL: MetadataViews.ExternalURL

        // Square-sized image to represent this collection.
        access(all) let squareImage: MetadataViews.Media

        // Banner-sized image for this collection, recommended to have a size near 1400x350.
        access(all) let bannerImage: MetadataViews.Media

        // Social links to reach this collection's social homepages.
        // Possible keys may be "instagram", "twitter", "discord", etc.
        access(all) let socials: {String: MetadataViews.ExternalURL}

        view init(
            name: String,
            description: String,
            externalURL: MetadataViews.ExternalURL,
            squareImage: MetadataViews.Media,
            bannerImage: MetadataViews.Media,
            socials: {String: MetadataViews.ExternalURL}
        ) {
            self.name = name
            self.description = description
            self.externalURL = externalURL
            self.squareImage = squareImage
            self.bannerImage = bannerImage
            self.socials = socials
        }
    }

    /// Helper to get NFTCollectionDisplay in a way that will return a typed
    /// Optional
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return A optional NFTCollection struct
    ///
    access(all) fun getNFTCollectionDisplay(_ viewResolver: &{ViewResolver.Resolver}) : NFTCollectionDisplay? {
        if let view = viewResolver.resolveView(Type<NFTCollectionDisplay>()) {
            if let v = view as? NFTCollectionDisplay {
                return v
            }
        }
        return nil
    }
    /// This view may be used by Cadence-native projects to define their
    /// contract- and token-level metadata according to EVM-compatible formats.
    /// Several ERC standards (e.g. ERC20, ERC721, etc.) expose name and symbol
    /// values to define assets as well as contract- & token-level metadata view
    /// `tokenURI(uint256)` and `contractURI()` methods. This view enables
    /// Cadence projects to define in their own contracts how they would like
    /// their metadata to be defined when bridged to EVM.
    ///
    access(all) struct EVMBridgedMetadata {

        /// The name of the asset
        ///
        access(all) let name: String

        /// The symbol of the asset
        ///
        access(all) let symbol: String

        /// The URI of the asset - this can either be contract-level or
        /// token-level URI depending on where the metadata is resolved. It
        /// is recommended to reference EVM metadata standards for how to best
        /// prepare your view's formatted value.
        ///
        /// For example, while you may choose to take advantage of onchain
        /// metadata, as is the case for most Cadence NFTs, you may also choose
        /// to represent your asset's metadata in IPFS and assign this value as
        /// an IPFSFile struct pointing to that IPFS file. Alternatively, you
        /// may serialize your NFT's metadata and assign it as a JSON string
        /// data URL representating the NFT's onchain metadata at the time this
        /// view is resolved.
        ///
        access(all) let uri: {File}

        init(name: String, symbol: String, uri: {File}) {
            self.name = name
            self.symbol = symbol
            self.uri = uri
        }
    }

    access(all) fun getEVMBridgedMetadata(_ viewResolver: &{ViewResolver.Resolver}) : EVMBridgedMetadata? {
        if let view = viewResolver.resolveView(Type<EVMBridgedMetadata>()) {
            if let v = view as? EVMBridgedMetadata {
                return v
            }
        }
        return nil
    }

}

import "FungibleToken"
import "MetadataViews"
import "ViewResolver"

/// This contract implements the metadata standard proposed
/// in FLIP-1087.
///
/// Ref: https://github.com/onflow/flips/blob/main/application/20220811-fungible-tokens-metadata.md
///
/// Structs and resources can implement one or more
/// metadata types, called views. Each view type represents
/// a different kind of metadata.
///
access(all) contract FungibleTokenMetadataViews {

    /// FTView wraps FTDisplay and FTVaultData, and is used to give a complete
    /// picture of a Fungible Token. Most Fungible Token contracts should
    /// implement this view.
    ///
    access(all) struct FTView {
        access(all) let ftDisplay: FTDisplay?
        access(all) let ftVaultData: FTVaultData?
        view init(
            ftDisplay: FTDisplay?,
            ftVaultData: FTVaultData?
        ) {
            self.ftDisplay = ftDisplay
            self.ftVaultData = ftVaultData
        }
    }

    /// Helper to get a FT view.
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return A FTView struct
    ///
    access(all) fun getFTView(viewResolver: &{ViewResolver.Resolver}): FTView {
        let maybeFTView = viewResolver.resolveView(Type<FTView>())
        if let ftView = maybeFTView {
            return ftView as! FTView
        }
        return FTView(
            ftDisplay: self.getFTDisplay(viewResolver),
            ftVaultData: self.getFTVaultData(viewResolver)
        )
    }

    /// View to expose the information needed to showcase this FT.
    /// This can be used by applications to give an overview and
    /// graphics of the FT.
    ///
    access(all) struct FTDisplay {
        /// The display name for this token.
        ///
        /// Example: "Flow"
        ///
        access(all) let name: String

        /// The abbreviated symbol for this token.
        ///
        /// Example: "FLOW"
        access(all) let symbol: String

        /// A description the provides an overview of this token.
        ///
        /// Example: "The FLOW token is the native currency of the Flow network."
        access(all) let description: String

        /// External link to a URL to view more information about the fungible token.
        access(all) let externalURL: MetadataViews.ExternalURL

        /// One or more versions of the fungible token logo.
        access(all) let logos: MetadataViews.Medias

        /// Social links to reach the fungible token's social homepages.
        /// Possible keys may be "instagram", "twitter", "discord", etc.
        access(all) let socials: {String: MetadataViews.ExternalURL}

        view init(
            name: String,
            symbol: String,
            description: String,
            externalURL: MetadataViews.ExternalURL,
            logos: MetadataViews.Medias,
            socials: {String: MetadataViews.ExternalURL}
        ) {
            self.name = name
            self.symbol = symbol
            self.description = description
            self.externalURL = externalURL
            self.logos = logos
            self.socials = socials
        }
    }

    /// Helper to get FTDisplay in a way that will return a typed optional.
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return An optional FTDisplay struct
    ///
    access(all) fun getFTDisplay(_ viewResolver: &{ViewResolver.Resolver}): FTDisplay? {
        if let maybeDisplayView = viewResolver.resolveView(Type<FTDisplay>()) {
            if let displayView = maybeDisplayView as? FTDisplay {
                return displayView
            }
        }
        return nil
    }

    /// View to expose the information needed store and interact with a FT vault.
    /// This can be used by applications to setup a FT vault with proper
    /// storage and public capabilities.
    ///
    access(all) struct FTVaultData {
        /// Path in storage where this FT vault is recommended to be stored.
        access(all) let storagePath: StoragePath

        /// Public path which must be linked to expose the public receiver capability.
        access(all) let receiverPath: PublicPath

        /// Public path which must be linked to expose the balance and resolver public capabilities.
        access(all) let metadataPath: PublicPath

        /// Type that should be linked at the `receiverPath`. This is a restricted type requiring
        /// the `FungibleToken.Receiver` interface.
        access(all) let receiverLinkedType: Type

        /// Type that should be linked at the `receiverPath`. This is a restricted type requiring
        /// the `ViewResolver.Resolver` interfaces.
        access(all) let metadataLinkedType: Type

        /// Function that allows creation of an empty FT vault that is intended
        /// to store the funds.
        access(all) let createEmptyVault: fun(): @{FungibleToken.Vault}

        view init(
            storagePath: StoragePath,
            receiverPath: PublicPath,
            metadataPath: PublicPath,
            receiverLinkedType: Type,
            metadataLinkedType: Type,
            createEmptyVaultFunction: fun(): @{FungibleToken.Vault}
        ) {
            pre {
                receiverLinkedType.isSubtype(of: Type<&{FungibleToken.Receiver}>()):
                    "Receiver public type <".concat(receiverLinkedType.identifier)
                    .concat("> must be a subtype of <").concat(Type<&{FungibleToken.Receiver}>().identifier)
                    .concat(">.")
                metadataLinkedType.isSubtype(of: Type<&{FungibleToken.Vault}>()):
                    "Metadata linked type <".concat(metadataLinkedType.identifier)
                    .concat("> must be a subtype of <").concat(Type<&{FungibleToken.Vault}>().identifier)
                    .concat(">.")
            }
            self.storagePath = storagePath
            self.receiverPath = receiverPath
            self.metadataPath = metadataPath
            self.receiverLinkedType = receiverLinkedType
            self.metadataLinkedType = metadataLinkedType
            self.createEmptyVault = createEmptyVaultFunction
        }
    }

    /// Helper to get FTVaultData in a way that will return a typed Optional.
    ///
    /// @param viewResolver: A reference to the resolver resource
    /// @return A optional FTVaultData struct
    ///
    access(all) fun getFTVaultData(_ viewResolver: &{ViewResolver.Resolver}): FTVaultData? {
        if let view = viewResolver.resolveView(Type<FTVaultData>()) {
            if let v = view as? FTVaultData {
                return v
            }
        }
        return nil
    }

    /// View to expose the total supply of the Vault's token
    access(all) struct TotalSupply {
        access(all) let supply: UFix64

        view init(totalSupply: UFix64) {
            self.supply = totalSupply
        }
    }
}

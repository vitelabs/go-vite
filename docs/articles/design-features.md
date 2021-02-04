# Security Algorithms in Vite
Asymmetric encryption, Hash and other security algorithms are the cornerstone of the blockchain. Among many implementations of security algorithms, how does Vite choose?

## A. Digital Signature Algorithm - [Ed25519](https://ed25519.cr.yp.to/)
### Background
The purpose of the digital signature is to create **non-repudiation**, a key characteristic of a wet signature (one we sign on paper contracts). The core feature of an effective digital signature is that the signer will be clearly identify based on the signed message, because the signed message can only be produced by relevant private key holders. Digital signature is legally enforceable in some countries, for instance, the Electronic Signature Act in USA and Signature Act in Germany.

The public key and the private key play critical roles in the process of signing with the private key, publishing the public key, and sending the message and its signed version for validation purposes. The pair tends to appear together, and is sort of inseparable like quantum entanglement. Public and private key are commonly generated through a special one-way function and a random number. The complexity of solving the one-way function f(x) = y is equivalent to that of solving polynomials, while solving the inverse function, f<sup>-1</sup>(y) = x, is exponential difficulty. The cost of calculating a private key from a public key exceeds the capability of any modern computing systems. The signature model is safe enough by now. However, if quantum computers are available in the future, then the current asymmetric encryption mechanism including RSA, DLP (Discrete Logarithm Problem) or ECC (Elliptical Curve Cryptography, expanded from DLP) will not be safe anymore. In other words, the attainment of quantum security will probably take as long as the invention of quantum computers.

The signature algorithm, on which the generation of account addresses and transactions is all based, is essential for the blockchain system. To select a signature algorithm, we need to evaluate the algorithm's security and performance, where security is the more important concern. Satoshi Nakamoto selected "ECDSA over secp256k1". Secp256k1 was a Koblitz curve defined by SECG.  Before Nakamoto's paper, the algorithm wasn't really used by anybody. "ECDSA over secp256k1" is designed transparently, while the "secp256r1" curve (the more mainstream P256 algorithm) used by NIST has weird parameters, which are widely recognized as the [back door](https://www.ams.org/notices/201402/rnoti-p190.pdf) arranged by NSA.  A series of subsequent incidents clearly proved the vision of Nakamoto's choice, which was picked up by Ethereum and EOS.

However, with the expiration of patent Ed25519, both the [core developers of Bitcoin](https://bitcointalk.org/index.php?topic=103172.msg1134832#msg1134832) and [Vitalik Buterin](https://blog.ethereum.org/2015/07/05/on-abstraction/) had talked about transfer to Ed25519.  Despite their preference for Ed25519 as indicated in the documents, the porting cost was too high to execute the transfer.  Whereas,  Ripple made such a migration in 2014 without any hesitation.

### Consideration of Security
Ed25519 was considered "safe" after being tested and reviewed by many independents and well-known security professionals, while secp256k1 was deemed "unsafe." [Here](https://safecurves.cr.yp.to/) is the reference.

### Consideration of Performance
To satisfy the industrial application requirements for high throughput, low latency and good scalability, Vite has designed several optimization schemes, including introducing the concept of "transaction decomposition" which means to decompose a transaction into a "Request" and a "Response."  As such, verification and confirmation can happen quite frequently, and therefore the performance of the whole system is especially sensitive to the speed of both signature algorithm and signature verification process.  The matter of this speed is especially important given Vite's high TPS target.  According to recent data, the performance of Ed25519 is several times faster than ECDSA over secp256k1 ([Reference: the benchmark built by Ripple](https://ripple.com/dev-blog/curves-with-a-twist/)). We believe that the speed boost will greatly improve the performance of Vite system. In addition, the signature of Ed25519 is slightly shorter than that of ECDSA, which reduces the pressure of the network transmission and the storage system.

For more detailed benchmark, check [Reference](https://bench.cr.yp.to/primitives-sign.html).

### Modification
Instead of using SHA2 in Ed25519, Vite uses Blake2b.

### Deficiency
Since the key space is non-linear, the system is unable to be compatible with Hierarchical Deterministic Key Derivation of BIP. 
***
## B. Hash algorithm - [Blake2b](https://blake2.net/)
### Background
The purpose of a hash algorithm is to generate a short, fixed-size summary from any long message. The Hash function is also one-way. But the difference between a hash function and the one-way function of asymmetric encryption system is that the latter function usually seeks a reverse solution, which is practically impossible while theoretically possible. That is to say, a public key includes all the information that is needed to generate the private key. However, a hash function is not theoretically reversible but is practically reversible. A hash has infinitely many source messages corresponding to it, and here is an example: If a hash function's output is n bits, then the amount of all the output will be 2<sup>n</sup>, but there is an infinite amount of possible inputs. According to the pigeonhole principle, If the input preimage is m * 2<sup>n</sup> bits long and the output of the hash function is evenly distributed, the probability of an X which allows Hash(X) = target being the preimage is 1/m(if not evenly distributed, the probability is even lower). In practice, however, the source text will not be too long, due to limited storage and computing power. Generally, if the input isn't very long, then the range of m will not be wide. So the X that makes Hash(X) = Target will probably be the real source message.

In the Vite system, the hash function is responsible for mining, data tamper-proofing, data protection etc. It is as fundamental as the signature algorithm, so we will seriously consider about the security and the performance of hash algorithm.

Security Consideration
Blake2 evolves from Blake. Blake lost when competing with keccak for SHA3 standard. The reason was that Blake and SHA2 are similar to each other, while NIST aimed at a Hash algorithm which was totally different from SHA2 standard.

>desire for SHA-3 to complement the existing SHA-2 algorithms … BLAKE is rather similar to SHA-2.

NIST appraised Blake quite highly:

>“BLAKE and Keccak have very large security margins.”。

So we assume that the level of security between Blake2 and keccak are similar.

### Performance Consideration
According to a massive amount of data, Blake2 beats any other Hash algorithm on generic CPUs(X86 ARM and etc.). For detailed performance, check [Reference](http://bench.cr.yp.to/results-sha3.html).

Another characteristics of Blake2 is that, the peak value of Blake2 algorithm designed by ASIC wouldn't be very high, which means the peak speed of mining is relatively low.
***
## C. Key Derivation Function - [scrypt](https://github.com/Tarsnap/scrypt)
### Background
In simple terms, Key Derivation Function is used to derive sub-private keys from a master private key. One example is to expand a short string to a required format through the KDF algorithm. KDF is similar to hash, with a difference of adding random variables to prevent being hacked by table-lookup attacks (e.g. rainbow tables). The scrypt we used is a memory-dependent KDF so every calculation takes up significant memory and time. So brute force attack is nearly impossible.

KDF is non-fundamental in our system. After converting the short variable-length keyword input by users through KDF and receiving keys of 256 bits, we use the keys along with AES-256-GCM algorithm to encrypt Ed25519 private keys so that those keys will be securely saved in personal computers.

### Reasons
From a technical perspective, there is no significant change between scrypt and argon2, which won 2015 Password Hashing Competition. But scrypt is more mature from a practical perspective because it was born earlier and used more widely. If there is no critical problem of argon2 in the next 2 to 3 years, we would consider keep using it.
***
## D. Relative terms
* "ECDSA"(Elliptic Curve Digital Signature Algorithm) is a digital signature algorithm using elliptic curve

* "secp256k1" is a parameter of ECDSA algorithm
    * "sec" is Standards for Efficient Cryptogrpahy introduced by SECG(Standards for Efficient Cryptography Group) 
    * "p" means the elliptic curve uses prime field
    * "256" means the length of the prime is 256 bits
    * "k" stands for Koblitz Curve
    * "1" means it is the first(only on actually) standard curve type

* Ed25519 is a [EdDSA](https://en.wikipedia.org/wiki/EdDSA) Signature algorithm using SHA512/256 and [Curve25519](https://en.wikipedia.org/wiki/Curve25519)

* NIST(National Institute of Standards and Technology) develops security standards like SHA3, P256 and etc..

* NSA stands for National Security Agency

* AES-256-GCM is an advanced encryption standard with 256-bits key in Galois/Counter Mode

import jwt from 'jsonwebtoken'
const EXPIRES_IN= 3600
export const createCustomSessionToken = (payload: any, secret: string): string => {
  try {
    // Tạo JWT với HMAC-SHA512
    const token = jwt.sign(payload, secret, {
      algorithm: 'HS256',
      header: { alg: 'HS256', typ: 'JWT' },
    })
    return token
  } catch (error) {
    console.error('Error creating custom SessionToken:', error)
    throw error
  }
}

const secret = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'
const token = createCustomSessionToken({accessKey: "AKIAN2EQBDNQKM27N6WK", iat: 1748340146,  exp: +Math.floor(+Date.now() / 1000) + EXPIRES_IN}, secret)
console.log(token)
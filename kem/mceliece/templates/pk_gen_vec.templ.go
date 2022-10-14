// +build ignore
// The previous line (and this one up to the warning below) is removed by the
// template generator.

// Code generated from pk_gen_vec.templ.go. DO NOT EDIT.

package {{.Pkg}}

import (
	"github.com/cloudflare/circl/kem/mceliece/internal"
)

{{ if .Is348864 }}
const exponent = 64
{{ else }}
const exponent = 128
{{ end }}

{{ if not .Is8192128 }}
func storeI(out []byte, in uint64, i int) {
	for j := 0; j < i; j++ {
		out[j] = byte((in >> (j * 8)) & 0xFF)
	}
}
{{end}}

func deBitSlicing(out []uint64, in [][gfBits]uint64) {
	for i := 0; i < (1 << gfBits); i++ {
		out[i] = 0
	}

	for i := 0; i < exponent; i++ {
		for j := gfBits - 1; j >= 0; j-- {
			for r := 0; r < 64; r++ {
				out[i*64+r] <<= 1
				out[i*64+r] |= (in[i][j] >> r) & 1
			}
		}
	}
}

func toBitslicing2x(out0 [][gfBits]uint64, out1 [][gfBits]uint64, in []uint64) {
	for i := 0; i < exponent; i++ {
		for j := gfBits - 1; j >= 0; j-- {
			for r := 63; r >= 0; r-- {
				out1[i][j] <<= 1
				out1[i][j] |= (in[i*64+r] >> (j + gfBits)) & 1
			}
		}

		for j := gfBits - 1; j >= 0; j-- {
			for r := 63; r >= 0; r-- {
				out0[i][gfBits-1-j] <<= 1
				out0[i][gfBits-1-j] |= (in[i*64+r] >> j) & 1
			}
		}
	}
}

{{ if or .Is6688128 .Is8192128 }}
func irrLoad(out [][gfBits]uint64, in []byte) {
	var (
		v0 uint64
		v1 uint64
	)
	irr := [sysT]uint16{}

	for i := 0; i < sysT; i++ {
		irr[i] = loadGf(in[i*2:])
	}

	for i := 0; i < gfBits; i++ {
		for j := 63; j >= 0; j-- {
			v0 <<= 1
			v1 <<= 1
			v0 |= uint64(irr[j]>>i) & 1
			v1 |= uint64(irr[j+64]>>i) & 1
		}

		out[0][i] = v0
		out[1][i] = v1
	}
}
{{else}}
{{ if .Is348864 }}
func irrLoad(out []uint64, in []byte) {
{{else}}
func irrLoad(out [][gfBits]uint64, in []byte) {
{{end}}
	irr := [sysT + 1]uint16{}

	for i := 0; i < sysT; i++ {
		irr[i] = loadGf(in[i*2:])
	}

	irr[sysT] = 1
	{{ if .Is348864 }}
	for i := 0; i < gfBits; i++ {
		out[i] = 0;
	}

	for i := sysT; i >= 0; i-- {
		for j := 0; j < gfBits; j++ {
			out[j] <<= 1;
			out[j] |= uint64(irr[i] >> j) & 1;
		}
	}
	{{ else }}
	v := [2]uint64{}
	for i := 0; i < gfBits; i++ {
		v[0] = 0
		v[1] = 0

		for j := 63; j >= 0; j-- {
			v[0] <<= 1
			v[0] |= uint64(irr[j]>>i) & 1
		}
		for j := sysT; j >= 64; j-- {
			v[1] <<= 1
			v[1] |= uint64(irr[j]>>i) & 1
		}

		out[0][i] = v[0]
		out[1][i] = v[1]
	}
	{{ end }}
}
{{end}}

// nolint:unparam
// Public key generation. Generate the public key `pk`,
// permutation `pi` and pivot element `pivots` based on the
// secret key `sk` and permutation `perm` provided.
// `pk` has `max(1 << GFBITS, SYS_N)` elements which is
// 4096 for mceliece348864 and 8192 for mceliece8192128.
// `sk` has `2 * SYS_T` elements and perm `1 << GFBITS`.
func pkGen(pk *[pkNRows * pkRowBytes]byte, irr []byte, perm *[1 << gfBits]uint32, pi *[1 << gfBits]int16, pivots *uint64) bool {
	const (
		nblocksH = (sysN + 63) / 64
		nblocksI = (pkNRows + 63) / 64
		{{ if or .Is348864 .Is6688128 }}
		blockIdx = nblocksI
		{{ else }}
		blockIdx = nblocksI - 1
		tail = pkNRows % 64
		{{ end }}
	)
	mat := [pkNRows][nblocksH]uint64{}
	ops := [pkNRows][nblocksI]uint64{}
	var mask uint64
	{{ if .Is348864 }}
	irrInt := [gfBits]uint64{}
	{{ else }}
	irrInt := [2][gfBits]uint64{}
	{{ end }}
	consts := [exponent][gfBits]uint64{}
	eval := [exponent][gfBits]uint64{}
	prod := [exponent][gfBits]uint64{}
	tmp := [gfBits]uint64{}
	list := [1 << gfBits]uint64{}
	{{ if .Is8192128 }}
	oneRow := [pkNCols / 64]uint64{}
	{{ else }}
	oneRow := [exponent]uint64{}
	{{ end }}

	// compute the inverses
	irrLoad(irrInt[:], irr)
	fft(eval[:], irrInt[:])
	vecCopy(prod[0][:], eval[0][:])
	for i := 1; i < exponent; i++ {
		vecMul(prod[i][:], prod[i-1][:], eval[i][:])
	}
	vecInv(tmp[:], prod[exponent-1][:])
	for i := exponent-2; i >= 0; i-- {
		vecMul(prod[i+1][:], prod[i][:], tmp[:])
		vecMul(tmp[:], tmp[:], eval[i+1][:])
	}
	vecCopy(prod[0][:], tmp[:])

	// fill matrix
	deBitSlicing(list[:], prod[:])
	for i := uint64(0); i < (1 << gfBits); i++ {
		list[i] <<= gfBits
		list[i] |= i
		list[i] |= (uint64(perm[i])) << 31
	}
	internal.UInt64Sort(list[:], 1<<gfBits)

	for i := 1; i < (1 << gfBits); i++ {
		if (list[i-1] >> 31) == (list[i] >> 31) {
			return false
		}
	}
	toBitslicing2x(consts[:], prod[:], list[:])

	for i := 0; i < (1 << gfBits); i++ {
		pi[i] = int16(list[i] & gfMask)
	}

	for j := 0; j < nblocksI; j++ {
		for k := 0; k < gfBits; k++ {
			mat[k][j] = prod[j][k]
		}
	}

	for i := 1; i < sysT; i++ {
		for j := 0; j < nblocksI; j++ {
			vecMul(prod[j][:], prod[j][:], consts[j][:])
			for k := 0; k < gfBits; k++ {
				mat[i*gfBits+k][j] = prod[j][k]
			}
		}
	}

	// gaussian elimination to obtain an upper triangular matrix
	// and keep track of the operations in ops
	{{ if .Is8192128 }}
	for i := 0; i < pkNRows/64; i++ {
		for j := 0; j < 64; j++ {
			row := i*64 + j
			for c := 0; c < pkNRows/64; c++ {
				ops[row][c] = 0
			}
		}
	}
	for i := 0; i < pkNRows/64; i++ {
		for j := 0; j < 64; j++ {
			row := i*64 + j
			ops[row][i] = 1
			ops[row][i] <<= j
		}
	}
	{{ else }}
	for i := 0; i < pkNRows; i++ {
		for j := 0; j < nblocksI; j++ {
			ops[i][j] = 0
		}
	}
	for i := 0; i < pkNRows; i++ {
		ops[i][i/64] = 1
		ops[i][i/64] <<= (i % 64)
	}

	{{ if not (or .Is348864 .Is6688128) }}
	column := [pkNRows]uint64{}
	for i := 0; i < pkNRows; i++ {
		column[i] = mat[i][blockIdx]
	}
	{{ end }}
	{{ end }}


	{{ if .Is8192128 }}
	for i := 0; i < pkNRows/64; i++ {
		for j := 0; j < 64; j++ {
			row := i*64 + j
	{{ else }}
	for row := 0; row < pkNRows; row++ {
		i := row >> 6
		j := row & 63
		{{ end }}

		for k := row + 1; k < pkNRows; k++ {
			mask = mat[row][i] >> j
			mask &= 1
			mask -= 1

			for c := 0; c < nblocksI; c++ {
				mat[row][c] ^= mat[k][c] & mask
				ops[row][c] ^= ops[k][c] & mask
			}
		}
		// return if not systematic
		if ((mat[row][i] >> j) & 1) == 0 {
			return false
		}

		for k := row + 1; k < pkNRows; k++ {
			mask = mat[k][i] >> j
			mask &= 1
			mask = -mask

			for c := 0; c < nblocksI; c++ {
				mat[k][c] ^= mat[row][c] & mask
				ops[k][c] ^= ops[row][c] & mask
			}
		}
	}
	{{ if .Is8192128 }}
	}
	{{ end }}


	// computing the lineaer map required to obatin the systematic form
	{{ if .Is8192128 }}
		for i := pkNRows/64 - 1; i >= 0; i-- {
			for j := 63; j >= 0; j-- {
				row := i*64 + j
				for k := 0; k < row; k++ {
					mask = mat[k][i] >> j
					mask &= 1
					mask = -mask

					for c := 0; c < pkNRows/64; c++ {
						ops[k][c] ^= ops[row][c] & mask
					}
				}
			}
		}
	{{else}}
	for row := pkNRows - 1; row >= 0; row-- {
		for k := 0; k < row; k++ {
			mask = mat[k][row/64] >> (row & 63)
			mask &= 1
			mask = -mask

			for c := 0; c < nblocksI; c++ {
				ops[k][c] ^= ops[row][c] & mask
			}
		}
	}
	{{end}}

	// apply the linear map to the non-systematic part
	for j := nblocksI; j < nblocksH; j++ {
		for k := 0; k < gfBits; k++ {
			mat[k][j] = prod[j][k]
		}
	}

	for i := 1; i < sysT; i++ {
		for j := nblocksI; j < nblocksH; j++ {
			vecMul(prod[j][:], prod[j][:], consts[j][:])
			for k := 0; k < gfBits; k++ {
				mat[i*gfBits+k][j] = prod[j][k]
			}
		}
	}

	pkp := pk[:]

	{{ if .Is8192128 }}
	for i := 0; i < pkNRows/64; i++ {
		for j := 0; j < 64; j++ {
			row := i*64 + j

			for k := 0; k < pkNCols/64; k++ {
				oneRow[k] = 0
			}

			for c := 0; c < pkNRows/64; c++ {
				for d := 0; d < 64; d++ {
					mask = ops[row][c] >> d
					mask &= 1
					mask = -mask

					for k := 0; k < pkNCols/64; k++ {
						oneRow[k] ^= mat[c*64+d][k+pkNRows/64] & mask
					}
				}
			}

			for k := 0; k < pkNCols/64; k++ {
				store8(pkp, oneRow[k])
				pkp = pkp[8:]
			}
		}
	}
	{{ else }}
	{{ if not (or .Is348864 .Is6688128) }}
	for i := 0; i < pkNRows; i++ {
		mat[i][blockIdx] = column[i]
	}
	{{end}}
	for row := 0; row < pkNRows; row++ {
		for k := 0; k < nblocksH; k++ {
			oneRow[k] = 0
		}

		for c := 0; c < pkNRows; c++ {
			mask = ops[row][c>>6] >> (c & 63)
			mask &= 1
			mask = -mask

			for k := blockIdx; k < nblocksH; k++ {
				oneRow[k] ^= mat[c][k] & mask
			}
		}

		var k int
		for k = blockIdx; k < nblocksH-1; k++ {
			{{ if not (or .Is348864 .Is6688128) }}
			oneRow[k] = (oneRow[k] >> tail) | (oneRow[k+1] << (64 - tail))
			{{end}}
			store8(pkp, oneRow[k])
			pkp = pkp[8:]
		}

		{{ if not (or .Is348864 .Is6688128) }}
		oneRow[k] >>= tail
		{{end}}
		storeI(pkp, oneRow[k], pkRowBytes%8)

		{{if .Is6960119}}
		pkp[(pkRowBytes%8)-1] &= (1 << (pkNCols % 8)) - 1 // removing redundant bits
		{{end}}

		pkp = pkp[pkRowBytes%8:]
	}
	{{ end }}
	return true
}

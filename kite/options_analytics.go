package kite

import (
	"errors"
	"math"
)

// normCDF calculates the cumulative distribution function for a standard normal distribution.
// P(X <= x) = 0.5 * (1 + erf(x / sqrt(2)))
func normCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

// normPDF calculates the probability density function for a standard normal distribution.
// f(x) = (1 / sqrt(2*pi)) * exp(-x^2 / 2)
func normPDF(x float64) float64 {
	return (1.0 / math.Sqrt(2*math.Pi)) * math.Exp(-0.5*x*x)
}

// blackScholes calculates the price of a European option (call or put).
func blackScholes(input OptionAnalyticsInput, volatility float64) float64 {
	// Ensure TimeToExpiry is positive to avoid math errors (e.g., NaN from Sqrt of negative, or division by zero)
	if input.TimeToExpiry <= 0 {
		// For zero time to expiry, option price is intrinsic value
		if input.IsCallOption {
			return math.Max(0, input.UnderlyingPrice-input.StrikePrice)
		}
		return math.Max(0, input.StrikePrice-input.UnderlyingPrice)
	}

	// Handle zero or negative volatility
	if volatility <= 0 {
		// If volatility is zero, the price is the discounted intrinsic value.
		var price float64
		if input.IsCallOption {
			price = input.UnderlyingPrice*math.Exp(-input.DividendYield*input.TimeToExpiry) - input.StrikePrice*math.Exp(-input.RiskFreeRate*input.TimeToExpiry)
		} else {
			price = input.StrikePrice*math.Exp(-input.RiskFreeRate*input.TimeToExpiry) - input.UnderlyingPrice*math.Exp(-input.DividendYield*input.TimeToExpiry)
		}
		return math.Max(0, price)
	}

	// d1 = (ln(S/K) + (r - q + 0.5*v^2)*T) / (v*sqrt(T))
	d1Numerator := math.Log(input.UnderlyingPrice/input.StrikePrice) + (input.RiskFreeRate-input.DividendYield+0.5*volatility*volatility)*input.TimeToExpiry
	d1Denominator := volatility * math.Sqrt(input.TimeToExpiry)
	if d1Denominator == 0 {
		if input.IsCallOption {
			return math.Max(0, input.UnderlyingPrice*math.Exp(-input.DividendYield*input.TimeToExpiry)-input.StrikePrice*math.Exp(-input.RiskFreeRate*input.TimeToExpiry))
		}
		return math.Max(0, input.StrikePrice*math.Exp(-input.RiskFreeRate*input.TimeToExpiry)-input.UnderlyingPrice*math.Exp(-input.DividendYield*input.TimeToExpiry))
	}
	d1 := d1Numerator / d1Denominator

	// d2 = d1 - v*sqrt(T)
	d2 := d1 - volatility*math.Sqrt(input.TimeToExpiry)

	var price float64
	if input.IsCallOption {
		// Call Price = S*exp(-qT)*N(d1) - K*exp(-rT)*N(d2)
		price = input.UnderlyingPrice*math.Exp(-input.DividendYield*input.TimeToExpiry)*normCDF(d1) - input.StrikePrice*math.Exp(-input.RiskFreeRate*input.TimeToExpiry)*normCDF(d2)
	} else {
		// Put Price = K*exp(-rT)*N(-d2) - S*exp(-qT)*N(-d1)
		price = input.StrikePrice*math.Exp(-input.RiskFreeRate*input.TimeToExpiry)*normCDF(-d2) - input.UnderlyingPrice*math.Exp(-input.DividendYield*input.TimeToExpiry)*normCDF(-d1)
	}
	return price
}

// blackScholesVega calculates the Vega of an option.
// Vega = S * N'(d1) * sqrt(T) * exp(-qT)
func blackScholesVega(input OptionAnalyticsInput, volatility float64) float64 {
	if input.TimeToExpiry <= 0 || volatility <= 0 || input.UnderlyingPrice <= 0 {
		return 0 // Vega is zero or undefined
	}

	d1Numerator := math.Log(input.UnderlyingPrice/input.StrikePrice) + (input.RiskFreeRate-input.DividendYield+0.5*volatility*volatility)*input.TimeToExpiry
	d1Denominator := volatility * math.Sqrt(input.TimeToExpiry)

	if d1Denominator == 0 {
		return 0
	}
	d1 := d1Numerator / d1Denominator

	// Vega = S * exp(-qT) * PDF(d1) * sqrt(T)
	vega := input.UnderlyingPrice * math.Exp(-input.DividendYield*input.TimeToExpiry) * normPDF(d1) * math.Sqrt(input.TimeToExpiry)
	return vega
}

// CalculateImpliedVolatility calculates the implied volatility of an option using the Newton-Raphson method.
func CalculateImpliedVolatility(input OptionAnalyticsInput, marketPrice float64) (float64, error) {
	const maxIterations = 100
	const tolerance = 1e-6 
	const minVolatility = 1e-4 
	const maxVolatility = 10.0 
	const verySmallVega = 1e-8 

	sigma := 0.5 // Initial guess

	for i := 0; i < maxIterations; i++ {
		calculatedPrice := blackScholes(input, sigma)
		diff := calculatedPrice - marketPrice

		if math.Abs(diff) < tolerance {
			return sigma, nil
		}

		vega := blackScholesVega(input, sigma)

		if math.Abs(vega) < verySmallVega {
			if math.Abs(diff) < tolerance*10 { 
				return sigma, nil
			}
			return 0, errors.New("vega is too small, implied volatility calculation unstable")
		}

		sigmaPrev := sigma
		sigma = sigma - diff/vega

		if sigma < minVolatility {
			sigma = minVolatility
		} else if sigma > maxVolatility {
			sigma = maxVolatility
		}
		
		if math.Abs(sigma-sigmaPrev) < tolerance*1e-2 && math.Abs(diff) > tolerance {
			// Break if stuck
		}
	}

	return 0, errors.New("implied volatility did not converge after maximum iterations")
}

// calculateDelta calculates the Delta of an option.
func calculateDelta(input OptionAnalyticsInput, volatility float64) float64 {
	if input.TimeToExpiry <= 0 || volatility <= 0 || input.UnderlyingPrice <= 0 {
		// Simplified Delta for edge cases (e.g., at expiry)
		if input.IsCallOption {
			if input.UnderlyingPrice > input.StrikePrice {
				return 1.0
			} else if input.UnderlyingPrice < input.StrikePrice {
				return 0.0
			}
			return 0.5 // At the money
		} else { // Put Option
			if input.UnderlyingPrice < input.StrikePrice {
				return -1.0
			} else if input.UnderlyingPrice > input.StrikePrice {
				return 0.0
			}
			return -0.5 // At the money
		}
	}

	d1Numerator := math.Log(input.UnderlyingPrice/input.StrikePrice) + (input.RiskFreeRate-input.DividendYield+0.5*volatility*volatility)*input.TimeToExpiry
	d1Denominator := volatility * math.Sqrt(input.TimeToExpiry)
	if d1Denominator == 0 { // Should be caught by earlier checks, but defensive
		return 0 // Or handle as above
	}
	d1 := d1Numerator / d1Denominator

	if input.IsCallOption {
		return math.Exp(-input.DividendYield*input.TimeToExpiry) * normCDF(d1)
	}
	// Put Option
	return math.Exp(-input.DividendYield*input.TimeToExpiry) * (normCDF(d1) - 1)
}

// calculateGamma calculates the Gamma of an option.
func calculateGamma(input OptionAnalyticsInput, volatility float64) float64 {
	if input.TimeToExpiry <= 0 || volatility <= 0 || input.UnderlyingPrice <= 0 {
		return 0 // Gamma is zero or undefined if T, vol, or S is zero/negative
	}

	d1Numerator := math.Log(input.UnderlyingPrice/input.StrikePrice) + (input.RiskFreeRate-input.DividendYield+0.5*volatility*volatility)*input.TimeToExpiry
	d1Denominator := volatility * math.Sqrt(input.TimeToExpiry)

	if d1Denominator == 0 {
		return 0
	}
	d1 := d1Numerator / d1Denominator

	// Gamma = (N'(d1) * exp(-qT)) / (S * v * sqrt(T))
	numerator := normPDF(d1) * math.Exp(-input.DividendYield*input.TimeToExpiry)
	denominator := input.UnderlyingPrice * volatility * math.Sqrt(input.TimeToExpiry)

	if denominator == 0 {
		return 0 // Or a very large number if appropriate, but 0 indicates no change, which is safer
	}
	return numerator / denominator
}

// calculateTheta calculates the Theta of an option (annualized).
// Note: Theta is typically negative. The value returned here is the decay per year.
// To get per day, divide by 365.
func calculateTheta(input OptionAnalyticsInput, volatility float64) float64 {
	if input.TimeToExpiry <= 0 || volatility <= 0 || input.UnderlyingPrice <= 0 {
		return 0 // Theta is complex at expiry or if other params are invalid
	}

	d1Numerator := math.Log(input.UnderlyingPrice/input.StrikePrice) + (input.RiskFreeRate-input.DividendYield+0.5*volatility*volatility)*input.TimeToExpiry
	d1Denominator := volatility * math.Sqrt(input.TimeToExpiry)
	if d1Denominator == 0 {
		return 0
	}
	d1 := d1Numerator / d1Denominator
	d2 := d1 - volatility*math.Sqrt(input.TimeToExpiry)

	term1 := -(input.UnderlyingPrice * normPDF(d1) * volatility * math.Exp(-input.DividendYield*input.TimeToExpiry)) / (2 * math.Sqrt(input.TimeToExpiry))

	if input.IsCallOption {
		term2 := -input.RiskFreeRate * input.StrikePrice * math.Exp(-input.RiskFreeRate*input.TimeToExpiry) * normCDF(d2)
		term3 := input.DividendYield * input.UnderlyingPrice * math.Exp(-input.DividendYield*input.TimeToExpiry) * normCDF(d1)
		return term1 + term2 + term3
	}
	// Put Option
	term2 := input.RiskFreeRate * input.StrikePrice * math.Exp(-input.RiskFreeRate*input.TimeToExpiry) * normCDF(-d2)
	term3 := -input.DividendYield * input.UnderlyingPrice * math.Exp(-input.DividendYield*input.TimeToExpiry) * normCDF(-d1)
	return term1 + term2 + term3
}

// calculateRho calculates the Rho of an option.
// Note: Rho is typically scaled by 0.01 to represent change per 1% interest rate move.
// The value returned here is the raw Rho.
func calculateRho(input OptionAnalyticsInput, volatility float64) float64 {
	if input.TimeToExpiry <= 0 || volatility <= 0 || input.UnderlyingPrice <= 0 {
		return 0 // Rho can be non-zero at expiry if r > 0, but simplified here
	}

	d1Numerator := math.Log(input.UnderlyingPrice/input.StrikePrice) + (input.RiskFreeRate-input.DividendYield+0.5*volatility*volatility)*input.TimeToExpiry
	d1Denominator := volatility * math.Sqrt(input.TimeToExpiry)

	if d1Denominator == 0 {
		return 0
	}
	d1 := d1Numerator / d1Denominator
	d2 := d1 - volatility*math.Sqrt(input.TimeToExpiry)

	if input.IsCallOption {
		// Call Rho = K * T * exp(-rT) * N(d2)
		return input.StrikePrice * input.TimeToExpiry * math.Exp(-input.RiskFreeRate*input.TimeToExpiry) * normCDF(d2)
	}
	// Put Rho = -K * T * exp(-rT) * N(-d2)
	return -input.StrikePrice * input.TimeToExpiry * math.Exp(-input.RiskFreeRate*input.TimeToExpiry) * normCDF(-d2)
}

// CalculateOptionAnalytics is a public facade function that computes Implied Volatility and all Option Greeks.
func CalculateOptionAnalytics(input OptionAnalyticsInput, marketPrice float64) (OptionAnalyticsOutput, error) {
	output := OptionAnalyticsOutput{}

	// 1. Calculate Implied Volatility
	iv, err := CalculateImpliedVolatility(input, marketPrice)
	if err != nil {
		return output, err // Return empty output and the error
	}
	output.ImpliedVolatility = iv

	// 2. Calculate Greeks using the calculated Implied Volatility
	delta := calculateDelta(input, iv)
	gamma := calculateGamma(input, iv)
	vega := blackScholesVega(input, iv) // This is the direct Vega calculation, not to be confused with IV's vega.
	theta := calculateTheta(input, iv)
	rho := calculateRho(input, iv)

	output.Greeks = OptionGreeks{
		Delta: delta,
		Gamma: gamma,
		Vega:  vega,
		Theta: theta,
		Rho:   rho,
	}

	return output, nil
}

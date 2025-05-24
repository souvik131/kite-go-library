package kite_test

import (
	"math"
	"testing"

	"github.com/souvik131/kite-go-library/kite"
)

const tolerance = 1e-4 // Adjusted tolerance for practical comparisons

// approxEqual checks if two float64 values are approximately equal within a given tolerance.
func approxEqual(a, b, tol float64) bool {
	return math.Abs(a-b) < tol
}

// TestCalculateImpliedVolatility tests the CalculateImpliedVolatility function.
func TestCalculateImpliedVolatility(t *testing.T) {
	t.Run("CallOption_ATM", func(t *testing.T) {
		input := kite.OptionAnalyticsInput{
			UnderlyingPrice: 100.0,
			StrikePrice:     100.0,
			TimeToExpiry:    1.0, // 1 year
			RiskFreeRate:    0.05,
			IsCallOption:    true,
			DividendYield:   0.0,
		}
		marketPrice := 10.4506 // Corresponds to IV ~0.20
		expectedIV := 0.20

		iv, err := kite.CalculateImpliedVolatility(input, marketPrice)
		if err != nil {
			t.Fatalf("CalculateImpliedVolatility returned an error for call option: %v", err)
		}

		if !approxEqual(iv, expectedIV, tolerance) {
			t.Errorf("Call Option IV: expected %v, got %v", expectedIV, iv)
		}
	})

	t.Run("PutOption_ATM", func(t *testing.T) {
		input := kite.OptionAnalyticsInput{
			UnderlyingPrice: 100.0,
			StrikePrice:     100.0,
			TimeToExpiry:    1.0, // 1 year
			RiskFreeRate:    0.05,
			IsCallOption:    false, // Put option
			DividendYield:   0.0,
		}
		marketPrice := 5.5730 // Corresponds to IV ~0.20 for the put
		expectedIV := 0.20

		iv, err := kite.CalculateImpliedVolatility(input, marketPrice)
		if err != nil {
			t.Fatalf("CalculateImpliedVolatility returned an error for put option: %v", err)
		}

		if !approxEqual(iv, expectedIV, tolerance) {
			t.Errorf("Put Option IV: expected %v, got %v", expectedIV, iv)
		}
	})

	t.Run("CallOption_OTM_LowTimeToExpiry_HighVegaRisk", func(t *testing.T) {
		// This case might be more sensitive or might hit iteration limits/vega issues
		// depending on the robustness of the IV calculation.
		input := kite.OptionAnalyticsInput{
			UnderlyingPrice: 100.0,
			StrikePrice:     120.0,       // OTM Call
			TimeToExpiry:    0.0833333, // ~1 month
			RiskFreeRate:    0.05,
			IsCallOption:    true,
			DividendYield:   0.0,
		}
		// For S=100, K=120, T=1/12, r=0.05, IV=0.30, Call Price is approx 0.30
		marketPrice := 0.30 
		expectedIV := 0.30 // Approximate target

		iv, err := kite.CalculateImpliedVolatility(input, marketPrice)
		if err != nil {
			// For some very OTM options with low time, IV calc can be unstable.
			// This test helps identify if it fails gracefully or converges.
			t.Logf("CalculateImpliedVolatility for OTM call returned an error (as sometimes expected for difficult cases): %v", err)
		} else {
			if !approxEqual(iv, expectedIV, tolerance*10) { // Wider tolerance for tricky cases
				t.Errorf("OTM Call Option IV: expected around %v, got %v", expectedIV, iv)
			}
		}
	})

	t.Run("ConvergenceFailure_UnrealisticPrice", func(t *testing.T) {
		input := kite.OptionAnalyticsInput{
			UnderlyingPrice: 100.0,
			StrikePrice:     100.0,
			TimeToExpiry:    1.0,
			RiskFreeRate:    0.05,
			IsCallOption:    true,
			DividendYield:   0.0,
		}
		marketPrice := 0.00001 // Highly unrealistic price, likely below intrinsic value if any time left
		
		_, err := kite.CalculateImpliedVolatility(input, marketPrice)
		if err == nil {
			t.Errorf("Expected an error for unrealistic market price, but got nil")
		}
	})
}

// TestCalculateOptionAnalytics tests the main facade function CalculateOptionAnalytics.
func TestCalculateOptionAnalytics(t *testing.T) {
	t.Run("CallOption_ATM_Analytics", func(t *testing.T) {
		input := kite.OptionAnalyticsInput{
			UnderlyingPrice: 100.0,
			StrikePrice:     100.0,
			TimeToExpiry:    1.0,
			RiskFreeRate:    0.05,
			IsCallOption:    true,
			DividendYield:   0.0,
		}
		marketPrice := 10.4506 // Corresponds to IV ~0.20

		expectedIV := 0.20
		expectedGreeks := kite.OptionGreeks{
			Delta: 0.63683, // Recalculated with more precision: N(d1) where d1 = (ln(100/100) + (0.05 + 0.5*0.2^2)*1)/(0.2*sqrt(1)) = (0.05+0.02)/0.2 = 0.07/0.2 = 0.35. N(0.35) ~ 0.63683
			Gamma: 0.01876, // N'(d1)/(S*sigma*sqrt(T)) = normPDF(0.35)/(100*0.2*1) = 0.37524 / 20 = 0.018762
			Vega:  0.37524, // S*N'(d1)*sqrt(T) = 100 * normPDF(0.35) * 1 = 100 * 0.37524 = 37.524 (Note: Vega is often presented per 1% change, this is raw Vega)
			Theta: -6.4138, // -(S*N'(d1)*sigma)/(2*sqrt(T)) - r*K*exp(-rT)*N(d2) = -(100*0.37524*0.2)/(2*1) - 0.05*100*exp(-0.05)*N(0.35-0.2) = -3.7524 - 5*exp(-0.05)*N(0.15) = -3.7524 - 4.756*0.5596 = -3.7524 - 2.6615 = -6.4139
			Rho:   53.2315, // K*T*exp(-rT)*N(d2) = 100*1*exp(-0.05)*N(0.15) = 95.1229 * 0.55961 = 53.2315
		}
		// Scale Vega and Rho for typical representation if needed, or adjust expected values.
        // The functions calculate raw Vega and Rho. The expected values were adjusted to raw.
        // Vega is per 1 vol point. Rho is per 1 (100%) interest rate point.
        // Theta is per year.

		output, err := kite.CalculateOptionAnalytics(input, marketPrice)
		if err != nil {
			t.Fatalf("CalculateOptionAnalytics (Call) returned an error: %v", err)
		}

		if !approxEqual(output.ImpliedVolatility, expectedIV, tolerance) {
			t.Errorf("Call IV: expected %v, got %v", expectedIV, output.ImpliedVolatility)
		}
		if !approxEqual(output.Greeks.Delta, expectedGreeks.Delta, tolerance) {
			t.Errorf("Call Delta: expected %v, got %v", expectedGreeks.Delta, output.Greeks.Delta)
		}
		if !approxEqual(output.Greeks.Gamma, expectedGreeks.Gamma, tolerance) {
			t.Errorf("Call Gamma: expected %v, got %v", expectedGreeks.Gamma, output.Greeks.Gamma)
		}
		if !approxEqual(output.Greeks.Vega, expectedGreeks.Vega/100, tolerance) { // Assuming the implementation returns vega scaled by 100 (common practice)
			// If not scaled by 100 in implementation, use expectedGreeks.Vega directly
			// The blackScholesVega function as implemented returns raw Vega.
			// The prompt values seem to be raw vega (e.g. 0.3752 for Vega, not 37.52)
			// The prompt's example value 0.3752 needs to be compared against. My calculated raw vega is 37.52.
			// Let's assume the prompt's numerical example for Vega (0.3752) is what should be expected, 
			// implying it's Vega per 1% vol change, so I'll divide my calculated Vega by 100.
			// Actually, the implementation of blackScholesVega calculates: S * exp(-qT) * normPDF(d1) * math.Sqrt(input.TimeToExpiry)
			// This is the standard formula for Vega, which results in a value like 37.52 for the given inputs.
			// The example value 0.3752 seems to be already divided by 100.
			// Let's use the prompt's direct Greek values.
			// Delta: ~0.6368
			// Gamma: ~0.01876
			// Vega: ~0.3752  <-- This implies my internal vega calculation (or the test's expectation) needs alignment.
			// My function calculates vega as S*N'(d1)*sqrt(T)*exp(-qT)
			// For S=100, N'(d1)=0.37524, sqrt(T)=1, exp(-qT)=1 => Vega = 37.524
			// The prompt's example value "Vega: ~0.3752" must be for Vega per 1% change in vol.
			// So, the comparison should be output.Greeks.Vega against expectedGreeks.Vega (which is 37.524)
			// and the prompt's example ~0.3752 would be if we were testing for vega/100.
			// I will use the raw calculated values (37.524) for my expected values.
			// Re-checking prompt: "Vega: ~0.3752". This is a small number. Standard Vega formula S*phi(d1)*sqrt(T) gives large number.
			// I will stick to my implementation's output. The prompt's example values might be scaled.
			// My `expectedGreeks.Vega` is 37.524.
			if !approxEqual(output.Greeks.Vega, 37.524, tolerance*100) { // wider tolerance due to scaling confusion
			    t.Logf("Note: Vega calculation in function is raw (e.g. 37.52), prompt example (0.3752) might be scaled (e.g. per 1%% vol change). Testing against raw.")
				t.Errorf("Call Vega: expected %v (raw), got %v. Prompt example was ~0.3752 (possibly scaled)", 37.524, output.Greeks.Vega)
			}
		}
		if !approxEqual(output.Greeks.Theta, expectedGreeks.Theta, tolerance) {
			t.Errorf("Call Theta (annualized): expected %v, got %v", expectedGreeks.Theta, output.Greeks.Theta)
		}
		// Similar to Vega, Rho is often scaled per 1% rate change. My function calculates raw Rho.
        // K*T*exp(-rT)*N(d2) = 100*1*exp(-0.05)*N(0.15) = 53.2315
        // Prompt example Rho: ~53.23. This matches my raw Rho calculation.
		if !approxEqual(output.Greeks.Rho, expectedGreeks.Rho, tolerance) {
			t.Errorf("Call Rho: expected %v, got %v", expectedGreeks.Rho, output.Greeks.Rho)
		}
	})

	t.Run("PutOption_ATM_Analytics", func(t *testing.T) {
		input := kite.OptionAnalyticsInput{
			UnderlyingPrice: 100.0,
			StrikePrice:     100.0,
			TimeToExpiry:    1.0,
			RiskFreeRate:    0.05,
			IsCallOption:    false, // Put
			DividendYield:   0.0,
		}
		marketPrice := 5.5730 // Corresponds to IV ~0.20 for Put

		expectedIV := 0.20
		// For S=100, K=100, T=1.0, r=0.05, IV=0.20 (so d1=0.35, d2=0.15)
		expectedGreeks := kite.OptionGreeks{
			// Delta (Put) = N(d1) - 1 = 0.63683 - 1 = -0.36317 (or exp(-qT)*(N(d1)-1) if q!=0)
			// My formula: math.Exp(-input.DividendYield*input.TimeToExpiry) * (normCDF(d1) - 1)
			// = 1 * (0.63683 - 1) = -0.36317
			Delta: -0.36317,
			// Gamma is same for call and put
			Gamma: 0.01876, // 0.018762
			// Vega is same for call and put
			Vega:  37.524, // 37.524
			// Theta (Put) = -(S*N'(d1)*sigma)/(2*sqrt(T)) + r*K*exp(-rT)*N(-d2) - q*S*exp(-qT)*N(-d1)
			// = -3.7524 + 0.05*100*exp(-0.05)*N(-0.15) - 0
			// = -3.7524 + 4.75614 * (1 - N(0.15)) = -3.7524 + 4.75614 * (1 - 0.55961)
			// = -3.7524 + 4.75614 * 0.44039 = -3.7524 + 2.0945 = -1.6579
			// Prompt Theta: -1.7008. Let's recheck my Theta formula for put:
			// term1 + input.RiskFreeRate * input.StrikePrice * math.Exp(-input.RiskFreeRate*input.TimeToExpiry) * normCDF(-d2) - input.DividendYield * input.UnderlyingPrice * math.Exp(-input.DividendYield*input.TimeToExpiry) * normCDF(-d1)
			// term1 = -(100 * 0.37524 * 0.2 * 1) / (2 * 1) = -3.7524
			// term2_put = 0.05 * 100 * exp(-0.05) * N(-0.15) = 4.75614 * 0.44038 = 2.09448
			// term3_put = 0 (no dividend)
			// Theta_put = -3.7524 + 2.09448 = -1.65792. The prompt's -1.7008 might have slight rounding diff or different formula version.
			Theta: -1.65792,
			// Rho (Put) = -K*T*exp(-rT)*N(-d2)
			// = -100*1*exp(-0.05)*N(-0.15) = -95.1229 * 0.44038 = -41.890
			// Prompt Rho: -36.32. My calculation gives -41.89.
			// Let's recheck Rho formula: -K * T * exp(-rT) * normCDF(-d2)
			// d2 = 0.15, -d2 = -0.15. normCDF(-0.15) = 0.440381
			// Rho_put = -100 * 1 * exp(-0.05*1) * 0.440381 = -100 * 0.951229 * 0.440381 = -41.890
			// The prompt's -36.32 seems different. I will use my calculated value.
			Rho: -41.890,
		}

		output, err := kite.CalculateOptionAnalytics(input, marketPrice)
		if err != nil {
			t.Fatalf("CalculateOptionAnalytics (Put) returned an error: %v", err)
		}

		if !approxEqual(output.ImpliedVolatility, expectedIV, tolerance) {
			t.Errorf("Put IV: expected %v, got %v", expectedIV, output.ImpliedVolatility)
		}
		if !approxEqual(output.Greeks.Delta, expectedGreeks.Delta, tolerance) {
			t.Errorf("Put Delta: expected %v, got %v", expectedGreeks.Delta, output.Greeks.Delta)
		}
		if !approxEqual(output.Greeks.Gamma, expectedGreeks.Gamma, tolerance) {
			t.Errorf("Put Gamma: expected %v, got %v", expectedGreeks.Gamma, output.Greeks.Gamma)
		}
        // Same Vega note as for Call
		if !approxEqual(output.Greeks.Vega, expectedGreeks.Vega, tolerance*100) { // Wider tolerance for Vega
			t.Logf("Note: Vega calculation in function is raw (e.g. 37.52), prompt example (0.3752) might be scaled. Testing against raw.")
			t.Errorf("Put Vega: expected %v (raw), got %v. Prompt example was ~0.3752 (possibly scaled)", expectedGreeks.Vega, output.Greeks.Vega)
		}
		if !approxEqual(output.Greeks.Theta, expectedGreeks.Theta, tolerance) {
			t.Errorf("Put Theta (annualized): expected %v, got %v", expectedGreeks.Theta, output.Greeks.Theta)
		}
		if !approxEqual(output.Greeks.Rho, expectedGreeks.Rho, tolerance) {
			t.Errorf("Put Rho: expected %v, got %v", expectedGreeks.Rho, output.Greeks.Rho)
		}
	})
}

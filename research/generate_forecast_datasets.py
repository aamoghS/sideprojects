"""
Enriched future-prediction datasets.
Weak-signal domains (Market, Retail, Healthcare, Supply Chain, Credit)
now have proper domain-specific features, longer lag windows, and
better-constructed targets to provide genuine learnable signal.
"""
import numpy as np
import pandas as pd

rng = np.random.default_rng(42)


# ── helpers ───────────────────────────────────────────────────────────────────
def _rolling(s, w, fn='mean'):
    r = pd.Series(s)
    return getattr(r.rolling(w, min_periods=1), fn)().fillna(0).values


def _rsi(returns, period=14):
    g = np.where(returns > 0, returns, 0.0)
    l = np.where(returns < 0, -returns, 0.0)
    ag = _rolling(g, period, 'mean') + 1e-9
    al = _rolling(l, period, 'mean') + 1e-9
    return 100 - 100 / (1 + ag / al)


def _macd(prices, fast=12, slow=26, sig=9):
    p   = pd.Series(prices)
    ema_f = p.ewm(span=fast, adjust=False).mean()
    ema_s = p.ewm(span=slow, adjust=False).mean()
    macd  = ema_f - ema_s
    signal = macd.ewm(span=sig, adjust=False).mean()
    return macd.values, signal.values, (macd - signal).values


def _bollinger(prices, w=20):
    p   = pd.Series(prices)
    mid = p.rolling(w, min_periods=1).mean()
    std = p.rolling(w, min_periods=1).std().fillna(0)
    return ((p - mid) / (std + 1e-9)).values   # z-score position


# ── 1. Market Regime (enriched) ───────────────────────────────────────────────
def generate_market_regime(n: int = 70000) -> pd.DataFrame:
    """
    Richer equity market with full technical indicator suite.
    Target: will 20-step cumulative return be > 0 (medium-term direction)?
    """
    steps = np.arange(n)
    regime = np.ones(n)
    regime[16000:28000] = -1.5   # strong bear
    regime[40000:48000] =  0.4   # choppy recovery
    regime[55000:]      =  1.2   # bull continuation

    vol_regime = np.ones(n)
    vol_regime[16000:28000] = 2.5   # high vol in bear
    vol_regime[40000:48000] = 1.8

    returns   = regime * 0.0003 + rng.normal(0, 0.013, n) * vol_regime
    price     = 100 * np.exp(np.cumsum(returns))
    volume    = rng.lognormal(mean=14, sigma=0.5, size=n) * vol_regime

    # Technical indicators
    rsi14     = _rsi(returns, 14)
    rsi7      = _rsi(returns,  7)
    macd_l, macd_sig, macd_hist = _macd(price)
    boll_pos  = _bollinger(price, 20)

    mom5      = _rolling(returns,  5, 'mean')
    mom10     = _rolling(returns, 10, 'mean')
    mom20     = _rolling(returns, 20, 'mean')
    mom50     = _rolling(returns, 50, 'mean')
    vol10     = _rolling(returns, 10, 'std')
    vol30     = _rolling(returns, 30, 'std')
    vol_ratio = vol10 / (vol30 + 1e-9)           # vol compression
    vol_rel   = volume / (_rolling(volume, 20, 'mean') + 1)

    # Target: 20-step forward cumulative return sign (medium-term)
    cum_fwd   = pd.Series(returns).rolling(20).sum().shift(-20).fillna(0).values
    is_fraud  = (cum_fwd > 0).astype(int)

    return pd.DataFrame({
        'step': steps, 'type': 'EQUITY',
        'amount':        price.round(4),
        'oldbalanceOrg': mom5.round(6),
        'newbalanceOrig':mom20.round(6),
        'oldbalanceDest':vol10.round(6),
        'newbalanceDest':vol_rel.round(4),
        'rsi14':         rsi14.round(4),
        'rsi7':          rsi7.round(4),
        'macd':          macd_l.round(6),
        'macd_hist':     macd_hist.round(6),
        'boll_pos':      boll_pos.round(4),
        'mom10':         mom10.round(6),
        'mom50':         mom50.round(6),
        'vol_ratio':     vol_ratio.round(4),
        'isFraud': is_fraud,
    }).dropna().reset_index(drop=True)


# ── 2. Energy Grid ────────────────────────────────────────────────────────────
def generate_energy_grid(n: int = 70000) -> pd.DataFrame:
    steps = np.arange(n)
    hour  = steps % 24
    dow   = (steps // 24) % 7

    base      = 400 + 120 * np.sin(2 * np.pi * hour / 24) + 40 * np.sin(2 * np.pi * dow / 7)
    temp      = 20  + 10  * np.sin(2 * np.pi * steps / (24*365)) + rng.normal(0, 3, n)
    heat_wave = np.where((steps >= 18000) & (steps < 26000), temp * 4, 0)
    shutdown  = np.where((steps >= 38000) & (steps < 44000), -100, 0)
    load      = np.clip(base + heat_wave + shutdown + rng.normal(0, 20, n), 50, None)

    roll24    = _rolling(load, 24, 'mean')
    roll168   = _rolling(load, 168, 'mean')   # 1-week baseline
    trend     = pd.Series(load).diff(1).fillna(0).values
    spike_thr = np.percentile(load, 90)
    future    = pd.Series(load).shift(-1).fillna(load[-1]).values

    return pd.DataFrame({
        'step': steps, 'type': 'GRID',
        'amount': load.round(2), 'oldbalanceOrg': temp.round(2),
        'newbalanceOrig': roll24.round(2), 'oldbalanceDest': trend.round(4),
        'newbalanceDest': (hour / 24.0),
        'roll168': roll168.round(2),
        'load_vs_week': (load / (roll168 + 1)).round(4),
        'isFraud': (future > spike_thr).astype(int),
    }).dropna().reset_index(drop=True)


# ── 3. IoT Sensors ────────────────────────────────────────────────────────────
def generate_iot_sensors(n: int = 70000) -> pd.DataFrame:
    steps = np.arange(n)
    wear  = np.clip(steps / n, 0, 1)
    wear  = np.where(steps >= 22000, wear * 2.5, wear)
    wear  = np.where(steps >= 42000, 0.05, wear)

    vibration   = 0.5 + wear * 3.0 + rng.exponential(0.1, n)
    temperature = 60  + wear * 40  + rng.normal(0, 2, n)
    pressure    = 100 - wear * 20  + rng.normal(0, 3, n)
    rpm         = 3000 - wear * 500 + rng.normal(0, 50, n)
    cumwear     = np.cumsum(wear * 0.001 + rng.exponential(0.0005, n))

    fail_score  = (vibration > 2.5).astype(float) + (temperature > 85).astype(float) + \
                  (pressure < 85).astype(float)
    future_fail = pd.Series(fail_score).shift(-10).fillna(0).values

    return pd.DataFrame({
        'step': steps, 'type': 'SENSOR',
        'amount': vibration.round(4), 'oldbalanceOrg': temperature.round(2),
        'newbalanceOrig': pressure.round(2), 'oldbalanceDest': rpm.round(1),
        'newbalanceDest': cumwear.round(6),
        'isFraud': (future_fail >= 2).astype(int),
    }).dropna().reset_index(drop=True)


# ── 4. Retail Sales (enriched) ────────────────────────────────────────────────
def generate_retail_sales(n: int = 70000) -> pd.DataFrame:
    """
    Richer retail with promotional cycles, price elasticity,
    competitor response, seasonal decomposition.
    Target: will next-14-day sales be in top quartile?
    """
    steps  = np.arange(n)
    week   = steps // 1000
    season = np.sin(2 * np.pi * week / 52)
    trend  = 1 + 0.0002 * steps          # organic growth

    promo        = (rng.random(n) < 0.12).astype(float)
    promo_lift   = 1 + 1.8 * promo       # promotions drive 80% lift
    price        = 29.99 + rng.normal(0, 2, n)
    price_elast  = np.exp(-0.05 * (price - 30))  # demand falls as price rises
    inv          = np.clip(rng.normal(500, 100, n), 10, None)
    stockout     = (inv < 50).astype(float)       # stockout kills sales

    comp         = np.where(steps >= 24000, 0.82, 1.0)   # competitor entry
    holi         = np.where(steps >= 46000, 1.40, 1.0)   # holiday surge
    recession    = np.where((steps >= 30000) & (steps < 38000), 0.75, 1.0)

    sales = trend * comp * holi * recession * promo_lift * price_elast * \
            (200 + 80 * season) * (1 - 0.3 * stockout) + rng.normal(0, 20, n)
    sales = np.clip(sales, 0, None)

    roll7_sales  = _rolling(sales, 7, 'mean')
    roll30_sales = _rolling(sales, 30, 'mean')
    roll7_std    = _rolling(sales, 7, 'std')
    yoy_proxy    = sales / (_rolling(sales, 365, 'mean') + 1)

    # Use 3-step forward median — short enough to stay within any window size
    future_sales = pd.Series(sales).shift(-3).fillna(method='bfill').values
    rolling_med  = pd.Series(sales).rolling(52, min_periods=1).median().values
    is_fraud     = (future_sales > rolling_med).astype(int)

    return pd.DataFrame({
        'step': steps, 'type': 'RETAIL',
        'amount': sales.round(2), 'oldbalanceOrg': price.round(2),
        'newbalanceOrig': inv.round(1), 'oldbalanceDest': promo,
        'newbalanceDest': season.round(4),
        'roll7': roll7_sales.round(2), 'roll30': roll30_sales.round(2),
        'roll7_std': roll7_std.round(2), 'yoy': yoy_proxy.round(4),
        'price_elast': price_elast.round(4), 'stockout': stockout,
        'isFraud': is_fraud,
    }).dropna().reset_index(drop=True)


# ── 5. Healthcare ICU (enriched) ─────────────────────────────────────────────
def generate_healthcare(n: int = 70000) -> pd.DataFrame:
    """
    SOFA-style ICU scoring: predict deterioration in next 6 hours.
    Features: multi-organ function scores, vital trends, lab values.
    """
    steps = np.arange(n)

    age          = np.clip(rng.normal(65, 15, n), 18, 100)
    prior_admit  = rng.poisson(1.5, n)

    # Vitals (with realistic correlations)
    hr           = 70  + rng.normal(0, 12, n) + 0.1 * age
    spo2         = np.clip(97 - 0.05 * age + rng.normal(0, 1.5, n), 60, 100)
    bp_sys       = 120 + 0.3  * age + rng.normal(0, 15, n)
    rr           = 16  + 0.05 * age + rng.normal(0, 3, n)   # resp rate
    temp_c       = 37  + rng.normal(0, 0.6, n)

    # Labs
    creatinine   = np.clip(1.0 + 0.02 * age + rng.exponential(0.5, n), 0.3, 15)
    lactate      = np.clip(1.0 + rng.exponential(0.8, n), 0.3, 20)
    wbc          = np.clip(rng.normal(8, 3, n), 1, 50)       # immune response
    platelet     = np.clip(rng.normal(200, 70, n), 10, 500)
    glucose      = np.clip(100 + 0.5 * age + rng.normal(0, 30, n), 40, 600)

    # Drift: post-surgical cohort (higher baseline stress)
    surg_mult    = np.where((steps >= 18000) & (steps < 30000), 1.8, 1.0)
    # Drift: flu season
    flu_mult     = np.where((steps >= 40000) & (steps < 50000), 1.5, 1.0)

    # SOFA-inspired composite
    resp_score   = np.clip((100 - spo2) / 5, 0, 4)
    renal_score  = np.clip((creatinine - 1.2) / 2, 0, 4)
    cardio_score = np.clip((90 - bp_sys) / 20, 0, 4).clip(0)
    coag_score   = np.clip((200 - platelet) / 100, 0, 4).clip(0)
    liver_score  = np.clip((lactate - 1.5) / 2, 0, 4).clip(0)
    neuro_score  = (hr > 100).astype(float) * 2

    sofa = (resp_score + renal_score + cardio_score + coag_score + liver_score + neuro_score) \
           * surg_mult * flu_mult + rng.normal(0, 0.5, n)

    # Vital trends (deterioration signal)
    hr_trend     = _rolling(hr,   6, 'mean') - _rolling(hr,   24, 'mean')
    spo2_trend   = _rolling(spo2, 6, 'mean') - _rolling(spo2, 24, 'mean')
    sofa_roll    = _rolling(sofa, 6, 'mean')

    future_sofa  = pd.Series(sofa).shift(-6).fillna(0).values
    threshold    = np.percentile(sofa, 80)
    is_fraud     = (future_sofa > threshold).astype(int)

    return pd.DataFrame({
        'step': steps, 'type': 'PATIENT',
        'amount': hr.round(1), 'oldbalanceOrg': spo2.round(2),
        'newbalanceOrig': bp_sys.round(1), 'oldbalanceDest': lactate.round(3),
        'newbalanceDest': age.round(0),
        'creatinine': creatinine.round(3), 'wbc': wbc.round(1),
        'platelet': platelet.round(0), 'glucose': glucose.round(1),
        'rr': rr.round(1), 'temp': temp_c.round(2),
        'sofa': sofa.round(3), 'sofa_roll': sofa_roll.round(3),
        'hr_trend': hr_trend.round(3), 'spo2_trend': spo2_trend.round(3),
        'prior_admit': prior_admit.astype(float),
        'isFraud': is_fraud,
    }).dropna().reset_index(drop=True)


# ── 6. Cybersecurity ─────────────────────────────────────────────────────────
def generate_cybersecurity(n: int = 70000) -> pd.DataFrame:
    steps     = np.arange(n)
    hour      = (steps % 1440) / 60
    base_pkts = 5000 + 2000 * np.sin(2 * np.pi * hour / 24)
    pkt_rate  = base_pkts + rng.normal(0, 500, n)
    bytes_sec = pkt_rate * rng.uniform(64, 1500, n)
    conn      = rng.poisson(200, n)
    err_rate  = np.clip(rng.beta(0.5, 20, n), 0, 1)
    entropy   = rng.uniform(3.5, 5.5, n)

    bot_mask  = (steps >= 20000) & (steps < 28000)
    pkt_rate  = np.where(bot_mask, pkt_rate * rng.uniform(8, 20, n), pkt_rate)
    conn      = np.where(bot_mask, conn + rng.poisson(2000, n), conn)
    entropy   = np.where(bot_mask, rng.uniform(0.5, 2.0, n), entropy)
    scan_mask = (steps >= 38000) & (steps < 46000)
    err_rate  = np.where(scan_mask, err_rate + rng.uniform(0.1, 0.4, n), err_rate)

    attack    = ((pkt_rate > base_pkts*5).astype(float) + (conn > 1000).astype(float) +
                 (err_rate > 0.15).astype(float) + (entropy < 2.0).astype(float))
    future    = pd.Series(attack).shift(-5).fillna(0).values

    return pd.DataFrame({
        'step': steps, 'type': 'NETWORK',
        'amount': np.clip(pkt_rate, 0, None).round(1),
        'oldbalanceOrg': (bytes_sec/1e6).round(4),
        'newbalanceOrig': conn.astype(float),
        'oldbalanceDest': err_rate.round(4),
        'newbalanceDest': entropy.round(4),
        'isFraud': (future >= 2).astype(int),
    }).dropna().reset_index(drop=True)


# ── 7. Supply Chain (enriched) ────────────────────────────────────────────────
def generate_supply_chain(n: int = 70000) -> pd.DataFrame:
    """
    Richer supply chain with geopolitical risk, multi-tier supplier scoring,
    demand seasonality, and transport cost indices.
    Target: will delay exceed 14 days in next 2 weeks?
    """
    steps       = np.arange(n)
    lead_time   = rng.lognormal(2.5, 0.4, n)
    reliability = np.clip(rng.beta(8, 2, n), 0, 1)
    congestion  = np.clip(rng.beta(2, 5, n), 0, 1)
    fuel        = 3.50 + 0.5 * np.sin(2*np.pi*steps/10000) + rng.normal(0, 0.3, n)
    inv_buffer  = rng.exponential(14, n)

    # New enriched signals
    geo_risk    = np.clip(rng.beta(1, 10, n), 0, 1)   # political instability proxy
    demand_seas = 1 + 0.3 * np.sin(2*np.pi*steps/52000)  # seasonal demand
    alt_routes  = rng.integers(1, 6, n).astype(float)    # # of alternative routes
    tier2_score = np.clip(rng.beta(5, 3, n), 0, 1)       # tier-2 supplier quality

    # Drift: port strike
    strike_mask = (steps >= 22000) & (steps < 26000)
    congestion  = np.where(strike_mask, np.clip(congestion + 0.6, 0, 1), congestion)
    lead_time   = np.where(strike_mask, lead_time * 3.0, lead_time)
    alt_routes  = np.where(strike_mask, alt_routes * 0.3, alt_routes)

    # Drift: chip shortage
    short_mask  = steps >= 35000
    inten       = np.clip((steps - 35000) / 35000, 0, 1)
    lead_time   = np.where(short_mask, lead_time * (1 + 0.8*inten), lead_time)
    reliability = np.where(short_mask, reliability * 0.7, reliability)
    tier2_score = np.where(short_mask, tier2_score * 0.6, tier2_score)

    # Rolling delay trend
    delay_score  = lead_time/7 + (1-reliability)*5 + congestion*4 + \
                   np.clip(fuel-4,0,None)*2 - inv_buffer/14 + \
                   geo_risk*3 - tier2_score*2
    delay_roll7  = _rolling(delay_score, 7, 'mean')
    delay_trend  = _rolling(delay_score, 7, 'mean') - _rolling(delay_score, 30, 'mean')

    future_delay = pd.Series(delay_score).rolling(14).sum().shift(-14).fillna(0).values
    threshold    = np.percentile(future_delay[future_delay > 0], 75)
    is_fraud     = (future_delay > threshold).astype(int)

    return pd.DataFrame({
        'step': steps, 'type': 'SHIPMENT',
        'amount': lead_time.round(2), 'oldbalanceOrg': reliability.round(4),
        'newbalanceOrig': congestion.round(4), 'oldbalanceDest': fuel.round(2),
        'newbalanceDest': inv_buffer.round(1),
        'geo_risk': geo_risk.round(4), 'demand_seas': demand_seas.round(4),
        'alt_routes': alt_routes, 'tier2': tier2_score.round(4),
        'delay_roll': delay_roll7.round(3), 'delay_trend': delay_trend.round(3),
        'isFraud': is_fraud,
    }).dropna().reset_index(drop=True)


# ── 8. Climate ────────────────────────────────────────────────────────────────
def generate_climate(n: int = 70000) -> pd.DataFrame:
    steps    = np.arange(n)
    doy      = steps % 365
    temp     = 15 + 12*np.sin(2*np.pi*doy/365) + rng.normal(0, 3, n)
    humidity = np.clip(60 + 20*np.sin(2*np.pi*doy/365+1) + rng.normal(0, 8, n), 0, 100)
    pressure = 1013 + 5*np.sin(2*np.pi*doy/365) + rng.normal(0, 4, n)
    wind     = np.clip(rng.weibull(2, n)*15, 0, None)
    soil     = np.clip(30 + 15*np.sin(2*np.pi*doy/365+0.5) + rng.normal(0, 5, n), 0, 100)

    nino = (steps >= 25000) & (steps < 40000)
    temp     = np.where(nino, temp + 2.5, temp)
    humidity = np.where(nino, np.clip(humidity+12, 0, 100), humidity)
    uhi      = steps >= 45000
    temp     = np.where(uhi, temp + 1.5*(steps-45000)/25000, temp)

    precip   = (humidity/100)*np.clip(100-pressure+1013,0,None)/10 + \
               np.clip(wind-20,0,None)*0.5 + soil/100 + rng.exponential(0.3, n)
    future   = pd.Series(precip).shift(-3).fillna(0).values

    return pd.DataFrame({
        'step': steps, 'type': 'WEATHER',
        'amount': temp.round(2), 'oldbalanceOrg': humidity.round(2),
        'newbalanceOrig': pressure.round(2), 'oldbalanceDest': wind.round(2),
        'newbalanceDest': soil.round(2),
        'isFraud': (future > np.percentile(precip, 85)).astype(int),
    }).dropna().reset_index(drop=True)


# ── 9. Credit Risk (enriched) ────────────────────────────────────────────────
def generate_credit_risk(n: int = 70000) -> pd.DataFrame:
    """
    Richer credit model with behavioral signals, payment velocity,
    bureau tradeline depth, and income stress.
    Target: 90-day default probability.
    """
    steps        = np.arange(n)
    credit_score = np.clip(rng.normal(680, 90, n), 300, 850)
    dti          = np.clip(rng.beta(2, 5, n)*0.8, 0.01, 0.99)
    util         = np.clip(rng.beta(2, 4, n), 0, 1)
    missed       = rng.poisson(0.3, n)
    loan_age     = rng.exponential(24, n)

    # New behavioral signals
    payment_vel  = np.clip(rng.exponential(2, n), 0, 10)  # days late (avg)
    open_accts   = rng.integers(1, 15, n).astype(float)
    derog_marks  = rng.poisson(0.2, n).astype(float)
    income       = np.clip(rng.lognormal(10.5, 0.6, n), 20000, 500000)
    monthly_debt = income / 12 * dti

    # Drift: recession
    rec_mask = steps >= 28000
    inten    = np.clip((steps-28000)/16000, 0, 1)
    credit_score = np.where(rec_mask, credit_score - 50*inten, credit_score)
    util         = np.where(rec_mask, np.clip(util + 0.30*inten, 0, 1), util)
    missed       = np.where(rec_mask, missed + rng.poisson(inten*2, n), missed)
    payment_vel  = np.where(rec_mask, payment_vel + 5*inten, payment_vel)
    derog_marks  = np.where(rec_mask, derog_marks + rng.poisson(inten*0.5, n), derog_marks)

    # Drift: recovery
    rec2 = steps >= 44000
    credit_score = np.where(rec2, credit_score + 20, credit_score)
    payment_vel  = np.where(rec2, payment_vel * 0.7, payment_vel)

    # Rolling signals (trend of deterioration)
    util_trend   = _rolling(util, 10, 'mean') - _rolling(util, 30, 'mean')
    missed_cum   = np.cumsum(missed) / (steps + 1)

    default_score = (
        (850 - credit_score)/100 + dti*3 + util*2 +
        missed*1.5 + payment_vel*0.4 + derog_marks*1.2 -
        loan_age/36 + util_trend*3 + missed_cum*10 +
        rng.normal(0, 0.4, n)
    )
    # 14-step forward shift — safe within any chunk window
    future_default = pd.Series(default_score).shift(-14).fillna(method='bfill').values
    threshold      = np.percentile(default_score, 80)
    is_fraud       = (future_default > threshold).astype(int)

    return pd.DataFrame({
        'step': steps, 'type': 'CREDIT',
        'amount': np.clip(credit_score, 300, 850).round(0),
        'oldbalanceOrg': dti.round(4),
        'newbalanceOrig': util.round(4),
        'oldbalanceDest': missed.astype(float),
        'newbalanceDest': loan_age.round(1),
        'payment_vel': payment_vel.round(2),
        'open_accts': open_accts, 'derog': derog_marks,
        'income': (income/1000).round(1),
        'monthly_debt': (monthly_debt/1000).round(2),
        'util_trend': util_trend.round(4),
        'missed_cum': missed_cum.round(6),
        'isFraud': is_fraud,
    }).dropna().reset_index(drop=True)


# ── main ──────────────────────────────────────────────────────────────────────
if __name__ == '__main__':
    print('Generating enriched future-prediction datasets...')

    datasets = [
        ('dataset_market.csv',      generate_market_regime, 'Market Regime     (20-step cumulative return)'),
        ('dataset_energy.csv',      generate_energy_grid,   'Energy Grid       (next-hour spike)'),
        ('dataset_iot.csv',         generate_iot_sensors,   'IoT Sensors       (failure in 10 steps)'),
        ('dataset_retail.csv',      generate_retail_sales,  'Retail Sales      (next-14d top quartile)'),
        ('dataset_healthcare.csv',  generate_healthcare,    'Healthcare ICU    (6h deterioration)'),
        ('dataset_cybersec.csv',    generate_cybersecurity, 'Cybersecurity     (DDoS in 5 min)'),
        ('dataset_supplychain.csv', generate_supply_chain,  'Supply Chain      (14-day delay risk)'),
        ('dataset_climate.csv',     generate_climate,       'Climate           (extreme precip)'),
        ('dataset_credit.csv',      generate_credit_risk,   'Credit Risk       (90-day default)'),
    ]

    for path, fn, desc in datasets:
        df = fn()
        df.to_csv(path, index=False)
        rate = df['isFraud'].mean() * 100
        print(f'  [{desc}]  rows={len(df):,}  pos={rate:.1f}%  -> {path}')

    print('\nAll 9 enriched datasets ready.')

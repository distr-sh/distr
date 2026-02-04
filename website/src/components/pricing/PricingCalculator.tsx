import {useEffect, useState} from 'preact/hooks';

export default function PricingCalculator() {
  const [internalUsers, setInternalUsers] = useState(1);
  const [externalCustomers, setExternalCustomers] = useState(1);
  const [billingCycle, setBillingCycle] = useState<'monthly' | 'yearly'>(
    'yearly',
  );
  const [currency, setCurrency] = useState<'$' | '€'>('$');

  // Base pricing per internal user and customer organization
  // Monthly billing prices
  const starterInternalUserPriceMonthly = 19;
  const starterExternalCustomerPriceMonthly = 29;
  const proInternalUserPriceMonthly = 29;
  const proExternalCustomerPriceMonthly = 69;

  // Yearly billing prices (billed monthly when on yearly plan)
  const starterInternalUserPriceYearly = 16;
  const starterExternalCustomerPriceYearly = 24;
  const proInternalUserPriceYearly = 24;
  const proExternalCustomerPriceYearly = 56;

  // Tiered pricing for Pro customer organizations (51+ licenses)
  const proExternalCustomerPriceMonthlyTier2 = 48;
  const proExternalCustomerPriceYearlyTier2 = 39;

  // Get current prices based on billing cycle
  const getStarterInternalUserPrice = () => {
    return billingCycle === 'monthly'
      ? starterInternalUserPriceMonthly
      : starterInternalUserPriceYearly;
  };

  const getStarterExternalCustomerPrice = () => {
    return billingCycle === 'monthly'
      ? starterExternalCustomerPriceMonthly
      : starterExternalCustomerPriceYearly;
  };

  const getProInternalUserPrice = () => {
    return billingCycle === 'monthly'
      ? proInternalUserPriceMonthly
      : proInternalUserPriceYearly;
  };

  const getProExternalCustomerPrice = () => {
    return billingCycle === 'monthly'
      ? proExternalCustomerPriceMonthly
      : proExternalCustomerPriceYearly;
  };

  // Current prices based on billing cycle
  const starterInternalUserPrice = getStarterInternalUserPrice();
  const starterExternalCustomerPrice = getStarterExternalCustomerPrice();
  const proInternalUserPrice = getProInternalUserPrice();
  const proExternalCustomerPrice = getProExternalCustomerPrice();

  // Plan limits
  const starterMaxExternalCustomers = 3;
  const proMaxExternalCustomers = 100;

  // Check if plans are within limits
  const isStarterAvailable = externalCustomers <= starterMaxExternalCustomers;
  const isProAvailable = externalCustomers <= proMaxExternalCustomers;

  // Check if more than 100 customers (Enterprise only)
  const isEnterpriseOnly = externalCustomers > proMaxExternalCustomers;

  // Check if plans should be blurred based on currency or customer count
  const shouldBlurStarter = !isStarterAvailable || currency === '€';
  const shouldBlurPro = !isProAvailable || currency === '€' || isEnterpriseOnly;

  // Force yearly billing when Enterprise only (more than 100 customers) or EUR selected
  const shouldForceYearly = isEnterpriseOnly || currency === '€';

  // Calculate total monthly prices (capped at plan limits)
  const calculateStarterMonthlyPrice = () => {
    const cappedCustomers = Math.min(
      externalCustomers,
      starterMaxExternalCustomers,
    );
    return (
      starterInternalUserPrice * internalUsers +
      starterExternalCustomerPrice * cappedCustomers
    );
  };

  const calculateProMonthlyPrice = () => {
    const cappedCustomers = Math.min(
      externalCustomers,
      proMaxExternalCustomers,
    );

    // Calculate tiered pricing for customer organizations
    let externalCustomerCost = 0;
    if (cappedCustomers <= 50) {
      // All customers at tier 1 price
      externalCustomerCost = proExternalCustomerPrice * cappedCustomers;
    } else {
      // First 50 at tier 1, remaining at tier 2
      const tier2Price =
        billingCycle === 'monthly'
          ? proExternalCustomerPriceMonthlyTier2
          : proExternalCustomerPriceYearlyTier2;
      externalCustomerCost =
        proExternalCustomerPrice * 50 + tier2Price * (cappedCustomers - 50);
    }

    return proInternalUserPrice * internalUsers + externalCustomerCost;
  };

  const starterMonthlyPrice = calculateStarterMonthlyPrice();
  const proMonthlyPrice = calculateProMonthlyPrice();

  // Calculate yearly total (monthly price * 12)
  const starterYearlyTotal = starterMonthlyPrice * 12;
  const proYearlyTotal = proMonthlyPrice * 12;

  // Helper function to round up and format price without commas
  const formatPrice = (price: number) => {
    return Math.ceil(price);
  };

  const decrementInternalUsers = () => {
    if (internalUsers > 1) {
      setInternalUsers(internalUsers - 1);
    }
  };

  const incrementInternalUsers = () => {
    setInternalUsers(internalUsers + 1);
  };

  const handleInternalUsersChange = (e: any) => {
    const value = e.target.value;
    if (value === '') {
      return; // Allow empty input temporarily
    }
    const numValue = parseInt(value, 10);
    if (!isNaN(numValue) && numValue >= 1) {
      setInternalUsers(numValue);
    }
  };

  const handleInternalUsersBlur = (e: any) => {
    const value = parseInt(e.target.value, 10);
    if (isNaN(value) || value < 1) {
      setInternalUsers(1);
    }
  };

  const decrementExternalCustomers = () => {
    if (externalCustomers > 1) {
      setExternalCustomers(externalCustomers - 1);
    }
  };

  const incrementExternalCustomers = () => {
    // Allow unlimited customers (no max limit)
    setExternalCustomers(externalCustomers + 1);
  };

  const handleExternalCustomersChange = (e: any) => {
    const value = e.target.value;
    if (value === '') {
      return; // Allow empty input temporarily
    }
    const numValue = parseInt(value, 10);
    if (!isNaN(numValue) && numValue >= 1) {
      setExternalCustomers(numValue);
      // Auto-switch to yearly when more than 100 customers
      if (numValue > proMaxExternalCustomers) {
        setBillingCycle('yearly');
      }
    }
  };

  const handleExternalCustomersBlur = (e: any) => {
    const value = parseInt(e.target.value, 10);
    if (isNaN(value) || value < 1) {
      setExternalCustomers(1);
    }
    // Allow values above 100, no cap
  };

  const handleCurrencyChange = (newCurrency: '$' | '€') => {
    setCurrency(newCurrency);
    // When EUR is selected, automatically switch to yearly billing
    if (newCurrency === '€') {
      setBillingCycle('yearly');
    }
  };

  // Auto-switch to yearly when customer count exceeds Pro limit
  useEffect(() => {
    if (isEnterpriseOnly && billingCycle === 'monthly') {
      setBillingCycle('yearly');
    }
  }, [isEnterpriseOnly, billingCycle]);

  return (
    <section>
      <div class="container mx-auto px-4 max-w-7xl">
        {/* Internal users, customer organizations, billing cycle and currency selection */}
        <div class="flex flex-col lg:flex-row justify-between items-start gap-4 mb-8 p-6 bg-white dark:bg-gray-900 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
          <div class="flex-1 flex flex-col items-start justify-between min-w-0">
            <div class="w-full min-h-[4rem] flex flex-col justify-start mb-5">
              <h3 class="mb-1 text-lg leading-tight">Internal Users</h3>
              <p class="mb-0 text-sm text-gray-600 dark:text-gray-400 leading-snug">
                Select how many internal users you need
              </p>
            </div>
            <div class="flex items-center justify-start gap-3 w-full">
              <button
                class="w-8 h-8 rounded-full border border-gray-400 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 text-xl flex items-center justify-center cursor-pointer transition-all hover:bg-accent-600 hover:text-white hover:border-accent-600 disabled:opacity-50 disabled:cursor-not-allowed leading-none p-0"
                onClick={decrementInternalUsers}
                disabled={internalUsers <= 1}>
                -
              </button>
              <input
                type="number"
                min="1"
                value={internalUsers}
                onInput={handleInternalUsersChange}
                onBlur={handleInternalUsersBlur}
                class="text-lg font-medium min-w-[4rem] w-16 text-center border border-gray-400 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 focus:outline-none focus:border-accent-600 focus:ring-2 focus:ring-accent-600/20"
                style="appearance: textfield; -moz-appearance: textfield; -webkit-appearance: none;"
              />
              <button
                class="w-8 h-8 rounded-full border border-gray-400 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 text-xl flex items-center justify-center cursor-pointer transition-all hover:bg-accent-600 hover:text-white hover:border-accent-600 leading-none p-0"
                onClick={incrementInternalUsers}>
                +
              </button>
            </div>
          </div>

          <div class="flex-1 flex flex-col items-start justify-between min-w-0">
            <div class="w-full min-h-[4rem] flex flex-col justify-start mb-5">
              <h3 class="mb-1 text-lg leading-tight">Customers</h3>
              <p class="mb-0 text-sm text-gray-600 dark:text-gray-400 leading-snug">
                Select how many customer organizations you have
              </p>
            </div>
            <div class="flex items-center justify-start gap-3 w-full">
              <button
                class="w-8 h-8 rounded-full border border-gray-400 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 text-xl flex items-center justify-center cursor-pointer transition-all hover:bg-accent-600 hover:text-white hover:border-accent-600 disabled:opacity-50 disabled:cursor-not-allowed leading-none p-0"
                onClick={decrementExternalCustomers}
                disabled={externalCustomers <= 1}>
                -
              </button>
              <input
                type="number"
                min="1"
                value={externalCustomers}
                onInput={handleExternalCustomersChange}
                onBlur={handleExternalCustomersBlur}
                class="text-lg font-medium min-w-[4rem] w-16 text-center border border-gray-400 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 focus:outline-none focus:border-accent-600 focus:ring-2 focus:ring-accent-600/20"
                style="appearance: textfield; -moz-appearance: textfield; -webkit-appearance: none;"
              />
              <button
                class="w-8 h-8 rounded-full border border-gray-400 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 text-xl flex items-center justify-center cursor-pointer transition-all hover:bg-accent-600 hover:text-white hover:border-accent-600 leading-none p-0"
                onClick={incrementExternalCustomers}>
                +
              </button>
            </div>
          </div>

          <div class="flex-1 flex flex-col items-start justify-between min-w-0">
            <div class="w-full min-h-[4rem] flex flex-col justify-start mb-5">
              <h3 class="mb-1 text-lg leading-tight">Billing</h3>
              <p class="mb-0 text-sm text-gray-600 dark:text-gray-400 leading-snug">
                Select your preferred billing schedule
              </p>
            </div>
            <div class="inline-flex bg-gray-200 dark:bg-gray-700 rounded-full p-1 w-full justify-center">
              <button
                class={`px-4 py-1.5 border-none rounded-3xl font-medium transition-all text-sm flex-1 text-center flex items-center justify-center gap-2 relative ${
                  shouldForceYearly
                    ? 'opacity-50 cursor-not-allowed text-gray-500 dark:text-gray-500'
                    : 'cursor-pointer text-gray-900 dark:text-white'
                } ${
                  billingCycle === 'monthly'
                    ? 'bg-white dark:bg-gray-800 shadow-md'
                    : 'bg-transparent'
                }`}
                onClick={() => {
                  if (!shouldForceYearly) {
                    setBillingCycle('monthly');
                  }
                }}
                disabled={shouldForceYearly}>
                Monthly
              </button>
              <button
                class={`px-4 py-1.5 border-none rounded-3xl cursor-pointer font-medium transition-all text-sm flex-1 text-center flex items-center justify-center gap-2 relative text-gray-900 dark:text-white ${
                  billingCycle === 'yearly'
                    ? 'bg-white dark:bg-gray-800 shadow-md'
                    : 'bg-transparent'
                }`}
                onClick={() => setBillingCycle('yearly')}>
                <span>Yearly</span>
                {!shouldForceYearly && (
                  <span class="inline-block bg-accent-600/20 text-accent-900 dark:text-accent-600 px-2 py-0.5 rounded-xl text-[0.7rem] font-semibold whitespace-nowrap leading-tight">
                    Save 20%
                  </span>
                )}
              </button>
            </div>
          </div>

          <div class="flex-1 flex flex-col items-start justify-between min-w-0">
            <div class="w-full min-h-[4rem] flex flex-col justify-start mb-5">
              <h3 class="mb-1 text-lg leading-tight">Currency</h3>
              <p class="mb-0 text-sm text-gray-600 dark:text-gray-400 leading-snug">
                Select your preferred billing currency
              </p>
            </div>
            <div class="inline-flex bg-gray-200 dark:bg-gray-700 rounded-full p-1 w-full justify-center">
              <button
                class={`px-4 py-1.5 border-none rounded-3xl cursor-pointer font-medium transition-all text-sm flex-1 text-center flex items-center justify-center gap-2 relative text-gray-900 dark:text-white ${
                  currency === '$'
                    ? 'bg-white dark:bg-gray-800 shadow-md'
                    : 'bg-transparent'
                }`}
                onClick={() => handleCurrencyChange('$')}>
                USD
              </button>
              <button
                class={`px-4 py-1.5 border-none rounded-3xl cursor-pointer font-medium transition-all text-sm flex-1 text-center flex items-center justify-center gap-2 relative text-gray-900 dark:text-white ${
                  currency === '€'
                    ? 'bg-white dark:bg-gray-800 shadow-md'
                    : 'bg-transparent'
                }`}
                onClick={() => handleCurrencyChange('€')}>
                EUR
              </button>
            </div>
          </div>
        </div>

        {/* Pricing cards */}
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Starter Plan */}
          <div
            class={`mt-10 min-h-[50rem] flex flex-col bg-white dark:bg-gray-900 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 transition-all ${
              shouldBlurStarter ? 'opacity-50 blur-sm pointer-events-none' : ''
            }`}>
            <div class="flex justify-center items-center flex-col p-6 text-center min-h-[18rem]">
              <h3 class="text-xl font-semibold">Starter</h3>
              <div class="text-4xl font-bold my-2">
                {currency}
                {formatPrice(starterMonthlyPrice)}
                <span class="text-base font-normal">/month</span>
              </div>
              <p class="mb-0 mt-2 text-sm">
                {currency}
                {formatPrice(starterInternalUserPrice)}/internal user +{' '}
                {currency}
                {formatPrice(starterExternalCustomerPrice)}/customer
                organization
                <br />
                <span class="text-xs text-gray-600 dark:text-gray-400 font-normal">
                  Up to {starterMaxExternalCustomers} customer organizations
                </span>
              </p>
              <p class="mb-0 mt-2 text-sm">
                {internalUsers}{' '}
                {internalUsers === 1 ? 'internal user' : 'internal users'} •{' '}
                {externalCustomers}{' '}
                {externalCustomers === 1
                  ? 'customer organization'
                  : 'customer organizations'}{' '}
                •
                {billingCycle === 'monthly'
                  ? ' Billed monthly'
                  : ` ${currency}${formatPrice(starterYearlyTotal)} billed yearly`}
              </p>
            </div>
            <hr class="mb-0 border-gray-200 dark:border-gray-700" />
            <div class="p-6 flex-grow">
              <h4 class="text-lg font-semibold mb-2 mt-0">
                First POCs + market validation
              </h4>
              <p class="text-sm leading-relaxed mb-6 mt-0 text-gray-700 dark:text-gray-300">
                Docker + agent installs to ship fast, iterate fast, and get
                customers updated instantly.
              </p>
              <ul class="list-none pl-0 mt-4 mb-0">
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  Up to 3 customer installs
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  1 deployment per customer
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  1 user per customer
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  Pre & Post install scripts
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  Customer Portal with installation instructions
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  Basic email support + onboarding
                </li>
              </ul>
              <div class="mt-6 mb-0 p-3 bg-gray-100 dark:bg-accent-600/15 border-l-4 border-accent-600 rounded text-sm leading-snug text-gray-800 dark:text-gray-200 font-medium italic">
                Fastest route to validate customer-install GTM
              </div>
            </div>
            <div class="p-6 pt-0">
              <a
                href="/onboarding/"
                class="inline-block w-full px-6 py-3 bg-accent-600 hover:bg-accent-700 border-2 border-accent-600 text-white font-medium rounded-lg text-center transition-colors no-underline">
                Start free trial →
              </a>
            </div>
          </div>

          {/* Pro Plan */}
          <div
            class={`mt-5 min-h-[55rem] flex flex-col bg-white dark:bg-gray-900 rounded-lg shadow-lg border-2 border-accent-600 relative pt-4 transition-all ${
              shouldBlurPro ? 'opacity-50 blur-sm pointer-events-none' : ''
            }`}>
            <div class="absolute top-0 left-0 right-0 bg-accent-600 text-white py-1.5 text-base font-medium z-10 shadow-md text-center w-full">
              Most popular
            </div>
            <div class="flex justify-center items-center flex-col p-6 text-center min-h-[9rem] pt-8">
              <h3 class="text-xl font-semibold">Pro</h3>
              <div class="text-4xl font-bold my-2">
                {currency}
                {formatPrice(proMonthlyPrice)}
                <span class="text-base font-normal">/month</span>
              </div>
              <p class="mb-0 mt-2 text-sm">
                {currency}
                {formatPrice(proInternalUserPrice)}/internal user
                {externalCustomers > 50 ? (
                  <>
                    <br />
                    {currency}
                    {formatPrice(proExternalCustomerPrice)}/customer
                    organization (first 50) + {currency}
                    {formatPrice(
                      billingCycle === 'monthly'
                        ? proExternalCustomerPriceMonthlyTier2
                        : proExternalCustomerPriceYearlyTier2,
                    )}
                    /customer organization (51+)
                  </>
                ) : (
                  <>
                    {' '}
                    + {currency}
                    {formatPrice(proExternalCustomerPrice)}/customer
                    organization
                  </>
                )}
                <br />
                <span class="text-xs text-gray-600 dark:text-gray-400 font-normal">
                  Up to {proMaxExternalCustomers} total customer organizations
                </span>
              </p>
              <p class="mb-0 mt-2 text-sm">
                {internalUsers}{' '}
                {internalUsers === 1 ? 'internal user' : 'internal users'} •{' '}
                {externalCustomers}{' '}
                {externalCustomers === 1
                  ? 'customer organization'
                  : 'customer organizations'}{' '}
                •
                {billingCycle === 'monthly'
                  ? ' Billed monthly'
                  : ` ${currency}${formatPrice(proYearlyTotal)} billed yearly`}
              </p>
            </div>
            <hr class="mb-0 border-gray-200 dark:border-gray-700" />
            <div class="p-6 flex-grow">
              <h4 class="text-lg font-semibold mb-2 mt-0">
                Rollout + operational scaling
              </h4>
              <p class="text-sm leading-relaxed mb-6 mt-0 text-gray-700 dark:text-gray-300">
                Platform teams deploy through Helm/Kubernetes. Version
                visibility, governance, and license control.
              </p>
              <ul class="list-none pl-0 mt-4 mb-0">
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  Up to 100 customer installs
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  3 deployments per customer
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  Up to 10 users per customer
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  SSO + RBAC
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  License Management
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  1TB container registry with FGAC
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  White Label
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  White glove onboarding + private Slack
                </li>
              </ul>
              <div class="mt-6 mb-0 p-3 bg-gray-100 dark:bg-accent-600/15 border-l-4 border-accent-600 rounded text-sm leading-snug text-gray-800 dark:text-gray-200 font-medium italic">
                Production-grade rollout engine — version control + identity
                control at scale
              </div>
            </div>
            <div class="p-6 pt-0">
              <a
                href="/onboarding/"
                class="inline-block w-full px-6 py-3 bg-accent-600 hover:bg-accent-700 text-white font-medium rounded-lg text-center transition-colors no-underline">
                Start free trial →
              </a>
            </div>
          </div>

          {/* Enterprise Plan */}
          <div class="mt-10 min-h-[50rem] flex flex-col bg-white dark:bg-gray-900 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700">
            <div class="flex justify-center items-center flex-col p-6 text-center min-h-[18rem]">
              <h3 class="text-xl font-semibold">Enterprise</h3>
              <div class="text-4xl font-bold my-2">Book Demo</div>
              <p class="mb-0 mt-2 text-sm"></p>
            </div>
            <hr class="mb-0 border-gray-200 dark:border-gray-700" />
            <div class="p-6 flex-grow">
              <h4 class="text-lg font-semibold mb-2 mt-0">
                Entire self-hosted lifecycle
              </h4>
              <p class="text-sm leading-relaxed mb-6 mt-0 text-gray-700 dark:text-gray-300">
                Distribute software, docs, support, workflows, licensing &
                billing — all in one platform.
              </p>
              <ul class="list-none pl-0 mt-4 mb-0">
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  Unlimited customer installs
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  Unlimited deployments
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  Unlimited internal users
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  Dedicated infrastructure + Full White Label
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  Automated workflows + advanced governance
                </li>
                <li class="pl-6 relative mb-3 before:content-['✓'] before:absolute before:left-0 before:text-green-600">
                  SLA + Dedicated Support Engineer
                </li>
              </ul>
              <div class="mt-6 mb-0 p-3 bg-gray-100 dark:bg-accent-600/15 border-l-4 border-accent-600 rounded text-sm leading-snug text-gray-800 dark:text-gray-200 font-medium italic">
                End-to-end commercial distribution suite — unified platform
              </div>
            </div>
            <div class="p-6 pt-0">
              <a
                href="/contact/"
                class="inline-block w-full px-6 py-3 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-900 dark:text-white font-medium rounded-lg text-center transition-colors no-underline">
                Contact Us →
              </a>
            </div>
          </div>
        </div>

        {/* Self-Hosting Info Box */}
        <div class="mt-20 w-2/3 mx-auto p-6 bg-gradient-to-r from-accent-600/10 to-accent-900/10 dark:from-accent-600/20 dark:to-accent-900/20 rounded-lg border-2 border-accent-600/30 dark:border-accent-600/50">
          <h3 class="text-2xl font-bold mb-3 text-gray-900 dark:text-white">
            Self-Hosting Distr?
          </h3>
          <p class="text-base leading-relaxed text-gray-700 dark:text-gray-300 mb-0">
            Use our{' '}
            <a
              href="https://github.com/distr-sh/distr"
              target="_blank"
              rel="noopener noreferrer"
              class="text-accent-600 hover:text-accent-900 dark:text-accent-600 dark:hover:text-accent-200 font-medium underline">
              community edition
            </a>{' '}
            with unlimited users and customer organizations for free with all
            Starter features included. For self-hosting our Pro edition, please{' '}
            <a
              href="/contact/"
              class="text-accent-600 hover:text-accent-900 dark:text-accent-600 dark:hover:text-accent-200 font-medium underline">
              contact us
            </a>
            .
          </p>
        </div>
      </div>
    </section>
  );
}

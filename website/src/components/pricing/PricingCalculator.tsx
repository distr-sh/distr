import {useState} from 'preact/hooks';

export default function PricingCalculator() {
  const [internalUsers, setInternalUsers] = useState(1);
  const [externalCustomers, setExternalCustomers] = useState(1);
  const [billingCycle, setBillingCycle] = useState<'monthly' | 'yearly'>(
    'yearly',
  );
  const [currency, setCurrency] = useState<'$' | '€'>('$');

  // Base monthly pricing per internal user and external customer
  const starterInternalUserPriceMonthly = 19;
  const starterExternalCustomerPriceMonthly = 29;
  const proInternalUserPriceMonthly = 29;
  const proExternalCustomerPriceMonthly = 69;

  // Yearly pricing is 20% less expensive than monthly (save 20%)
  const yearlyDiscount = 0.8;

  // Get current prices based on billing cycle
  const getStarterInternalUserPrice = () => {
    return billingCycle === 'monthly'
      ? starterInternalUserPriceMonthly
      : starterInternalUserPriceMonthly * yearlyDiscount;
  };

  const getStarterExternalCustomerPrice = () => {
    return billingCycle === 'monthly'
      ? starterExternalCustomerPriceMonthly
      : starterExternalCustomerPriceMonthly * yearlyDiscount;
  };

  const getProInternalUserPrice = () => {
    return billingCycle === 'monthly'
      ? proInternalUserPriceMonthly
      : proInternalUserPriceMonthly * yearlyDiscount;
  };

  const getProExternalCustomerPrice = () => {
    return billingCycle === 'monthly'
      ? proExternalCustomerPriceMonthly
      : proExternalCustomerPriceMonthly * yearlyDiscount;
  };

  // Current prices based on billing cycle
  const starterInternalUserPrice = getStarterInternalUserPrice();
  const starterExternalCustomerPrice = getStarterExternalCustomerPrice();
  const proInternalUserPrice = getProInternalUserPrice();
  const proExternalCustomerPrice = getProExternalCustomerPrice();

  // Plan limits
  const starterMaxExternalCustomers = 3;
  const proMaxExternalCustomers = 50;

  // Check if plans are within limits
  const isStarterAvailable = externalCustomers <= starterMaxExternalCustomers;
  const isProAvailable = externalCustomers <= proMaxExternalCustomers;

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
    return (
      proInternalUserPrice * internalUsers +
      proExternalCustomerPrice * cappedCustomers
    );
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
    }
  };

  const handleExternalCustomersBlur = (e: any) => {
    const value = parseInt(e.target.value, 10);
    if (isNaN(value) || value < 1) {
      setExternalCustomers(1);
    }
  };

  return (
    <section>
      <div class="container mx-auto px-4 max-w-7xl">
        {/* Internal users, external customers, billing cycle and currency selection */}
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
                class="w-8 h-8 rounded-full border border-gray-400 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 text-xl flex items-center justify-center cursor-pointer transition-all hover:bg-[#00b5eb] hover:text-white hover:border-[#00b5eb] disabled:opacity-50 disabled:cursor-not-allowed leading-none p-0"
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
                class="text-lg font-medium min-w-[4rem] w-16 text-center border border-gray-400 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 focus:outline-none focus:border-[#00b5eb] focus:ring-2 focus:ring-[#00b5eb]/20"
                style="appearance: textfield; -moz-appearance: textfield; -webkit-appearance: none;"
              />
              <button
                class="w-8 h-8 rounded-full border border-gray-400 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 text-xl flex items-center justify-center cursor-pointer transition-all hover:bg-[#00b5eb] hover:text-white hover:border-[#00b5eb] leading-none p-0"
                onClick={incrementInternalUsers}>
                +
              </button>
            </div>
          </div>

          <div class="flex-1 flex flex-col items-start justify-between min-w-0">
            <div class="w-full min-h-[4rem] flex flex-col justify-start mb-5">
              <h3 class="mb-1 text-lg leading-tight">External Customers</h3>
              <p class="mb-0 text-sm text-gray-600 dark:text-gray-400 leading-snug">
                Select how many external customers you have
              </p>
            </div>
            <div class="flex items-center justify-start gap-3 w-full">
              <button
                class="w-8 h-8 rounded-full border border-gray-400 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 text-xl flex items-center justify-center cursor-pointer transition-all hover:bg-[#00b5eb] hover:text-white hover:border-[#00b5eb] disabled:opacity-50 disabled:cursor-not-allowed leading-none p-0"
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
                class="text-lg font-medium min-w-[4rem] w-16 text-center border border-gray-400 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 focus:outline-none focus:border-[#00b5eb] focus:ring-2 focus:ring-[#00b5eb]/20"
                style="appearance: textfield; -moz-appearance: textfield; -webkit-appearance: none;"
              />
              <button
                class="w-8 h-8 rounded-full border border-gray-400 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 text-xl flex items-center justify-center cursor-pointer transition-all hover:bg-[#00b5eb] hover:text-white hover:border-[#00b5eb] leading-none p-0"
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
                class={`px-4 py-1.5 border-none rounded-3xl cursor-pointer font-medium transition-all text-sm flex-1 text-center flex items-center justify-center gap-2 relative text-gray-900 dark:text-white ${
                  billingCycle === 'monthly'
                    ? 'bg-white dark:bg-gray-800 shadow-md'
                    : 'bg-transparent'
                }`}
                onClick={() => setBillingCycle('monthly')}>
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
                <span class="inline-block bg-[#00b5eb]/20 text-[#174c76] dark:text-[#00b5eb] px-2 py-0.5 rounded-xl text-[0.7rem] font-semibold whitespace-nowrap leading-tight">
                  Save 20%
                </span>
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
                onClick={() => setCurrency('$')}>
                USD
              </button>
              <button
                class={`px-4 py-1.5 border-none rounded-3xl cursor-pointer font-medium transition-all text-sm flex-1 text-center flex items-center justify-center gap-2 relative text-gray-900 dark:text-white ${
                  currency === '€'
                    ? 'bg-white dark:bg-gray-800 shadow-md'
                    : 'bg-transparent'
                }`}
                onClick={() => setCurrency('€')}>
                EUR
              </button>
            </div>
          </div>
        </div>

        {/* Pricing cards */}
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Starter Plan */}
          <div
            class={`mt-10 min-h-[55rem] flex flex-col bg-white dark:bg-gray-900 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 transition-all ${
              !isStarterAvailable
                ? 'opacity-50 blur-sm pointer-events-none'
                : ''
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
                {formatPrice(starterExternalCustomerPrice)}/external customer
                <br />
                <span class="text-xs text-gray-600 dark:text-gray-400 font-normal">
                  Up to {starterMaxExternalCustomers} external customers
                </span>
              </p>
              <p class="mb-0 mt-2 text-sm">
                {internalUsers}{' '}
                {internalUsers === 1 ? 'internal user' : 'internal users'} •{' '}
                {externalCustomers}{' '}
                {externalCustomers === 1
                  ? 'external customer'
                  : 'external customers'}{' '}
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
              <div class="mt-6 mb-0 p-3 bg-gray-100 dark:bg-[#00b5eb]/15 border-l-4 border-[#00b5eb] rounded text-sm leading-snug text-gray-800 dark:text-gray-200 font-medium italic">
                Fastest route to validate customer-install GTM
              </div>
            </div>
            <div class="p-6 pt-0">
              <a
                href="https://signup.distr.sh/"
                class="inline-block w-full px-6 py-3 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-900 dark:text-white font-medium rounded-lg text-center transition-colors no-underline">
                Get Started →
              </a>
            </div>
          </div>

          {/* Pro Plan */}
          <div
            class={`mt-5 min-h-[60rem] flex flex-col bg-white dark:bg-gray-900 rounded-lg shadow-lg border-2 border-[#00b5eb] relative pt-4 transition-all ${
              !isProAvailable ? 'opacity-50 blur-sm pointer-events-none' : ''
            }`}>
            <div class="absolute top-0 left-0 right-0 bg-[#00b5eb] text-white py-1.5 text-base font-medium z-10 shadow-md text-center w-full">
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
                {formatPrice(proInternalUserPrice)}/internal user + {currency}
                {formatPrice(proExternalCustomerPrice)}/external customer
                <br />
                <span class="text-xs text-gray-600 dark:text-gray-400 font-normal">
                  Up to {proMaxExternalCustomers} external customers
                </span>
              </p>
              <p class="mb-0 mt-2 text-sm">
                {internalUsers}{' '}
                {internalUsers === 1 ? 'internal user' : 'internal users'} •{' '}
                {externalCustomers}{' '}
                {externalCustomers === 1
                  ? 'external customer'
                  : 'external customers'}{' '}
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
                  Up to 50 customer installs
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
              <div class="mt-6 mb-0 p-3 bg-gray-100 dark:bg-[#00b5eb]/15 border-l-4 border-[#00b5eb] rounded text-sm leading-snug text-gray-800 dark:text-gray-200 font-medium italic">
                Production-grade rollout engine — version control + identity
                control at scale
              </div>
            </div>
            <div class="p-6 pt-0">
              <a
                href="https://cal.glasskube.com/team/gk/distr-pro-early-access"
                class="inline-block w-full px-6 py-3 bg-[#00b5eb] hover:bg-[#174c76] text-white font-medium rounded-lg text-center transition-colors no-underline">
                Get early access →
              </a>
            </div>
          </div>

          {/* Enterprise Plan */}
          <div class="mt-10 min-h-[55rem] flex flex-col bg-white dark:bg-gray-900 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700">
            <div class="flex justify-center items-center flex-col p-6 text-center min-h-[18rem]">
              <h3 class="text-xl font-semibold">Enterprise</h3>
              <div class="text-4xl font-bold my-2">Get a Demo</div>
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
              <div class="mt-6 mb-0 p-3 bg-gray-100 dark:bg-[#00b5eb]/15 border-l-4 border-[#00b5eb] rounded text-sm leading-snug text-gray-800 dark:text-gray-200 font-medium italic">
                End-to-end commercial distribution suite — unified platform
              </div>
            </div>
            <div class="p-6 pt-0">
              <a
                href="https://signup.distr.sh/"
                class="inline-block w-full px-6 py-3 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-900 dark:text-white font-medium rounded-lg text-center transition-colors no-underline">
                Contact Us →
              </a>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

import {useEffect, useState} from 'preact/hooks';

interface WhitePaperRequest {
  firstName: string;
  lastName: string;
  email: string;
  phone: string;
  jobTitle: string;
  companyName: string;
}

interface WhitePaperFormProps {
  formsServerBaseUrl: string;
  reCaptchaKeyV2: string;
}

// Load reCAPTCHA script
function loadRecaptchaScript() {
  if (typeof window === 'undefined') {
    return;
  }

  const elementId = 'recaptcha-script';
  if (document.getElementById(elementId) === null) {
    const script = document.createElement('script');
    script.src = 'https://www.google.com/recaptcha/api.js';
    script.id = elementId;
    script.async = true;
    script.defer = true;
    document.head.appendChild(script);
  }
}

export default function WhitePaperForm({
  formsServerBaseUrl,
  reCaptchaKeyV2,
}: WhitePaperFormProps) {
  const [formData, setFormData] = useState<WhitePaperRequest>({
    firstName: '',
    lastName: '',
    email: '',
    phone: '',
    jobTitle: '',
    companyName: '',
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [reCaptchaVerified, setReCaptchaVerified] = useState(false);
  const [reCaptchaToken, setReCaptchaToken] = useState('');

  // Load reCAPTCHA script and set up callback on component mount
  useEffect(() => {
    loadRecaptchaScript();

    // Set up global callback for reCAPTCHA
    (window as any).onWhitePaperRecaptchaCallback = (token: string) => {
      setReCaptchaVerified(true);
      setReCaptchaToken(token);
    };

    return () => {
      delete (window as any).onWhitePaperRecaptchaCallback;
    };
  }, []);

  const handleSubmit = async (event: Event) => {
    event.preventDefault();

    if (!reCaptchaVerified) {
      alert('Please complete the reCAPTCHA verification.');
      return;
    }

    // Get the reCAPTCHA token
    let token = reCaptchaToken;
    if (!token && (window as any).grecaptcha) {
      token = (window as any).grecaptcha.getResponse();
    }

    if (!token) {
      alert('Please complete the reCAPTCHA verification.');
      return;
    }

    setIsSubmitting(true);

    try {
      const response = await fetch(`${formsServerBaseUrl}/white-paper`, {
        method: 'POST',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ...formData,
          reCaptchaToken: token,
        }),
      });

      if (response.ok) {
        alert(
          'You will receive the white paper immediately via email. Thank you!',
        );
        // Reset form
        setFormData({
          firstName: '',
          lastName: '',
          email: '',
          phone: '',
          jobTitle: '',
          companyName: '',
        });

        // Reset reCAPTCHA
        setReCaptchaVerified(false);
        setReCaptchaToken('');
        if ((window as any).grecaptcha) {
          (window as any).grecaptcha.reset();
        }
      } else {
        throw new Error('Failed to submit form');
      }
    } catch (err) {
      alert(err instanceof Error ? err.message : 'An error occurred');
      console.error(err);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div class="w-full">
      <div class="text-center mb-6">
        <h2 class="text-2xl md:text-3xl font-bold mb-3 text-gray-900 dark:text-white">
          Get the white paper
        </h2>
        <p class="text-gray-600 dark:text-gray-400">
          You will receive the white paper immediately via email.
        </p>
      </div>
      <form onSubmit={handleSubmit} class="flex flex-col gap-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="firstName"
              class="block text-sm font-medium mb-2 text-gray-900 dark:text-white">
              First Name*
            </label>
            <input
              type="text"
              id="firstName"
              name="firstName"
              placeholder="First name"
              value={formData.firstName}
              onInput={e =>
                setFormData({...formData, firstName: e.currentTarget.value})
              }
              required
              class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors"
            />
          </div>
          <div>
            <label
              htmlFor="lastName"
              class="block text-sm font-medium mb-2 text-gray-900 dark:text-white">
              Last Name*
            </label>
            <input
              type="text"
              id="lastName"
              name="lastName"
              placeholder="Last name"
              value={formData.lastName}
              onInput={e =>
                setFormData({...formData, lastName: e.currentTarget.value})
              }
              required
              class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors"
            />
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="email"
              class="block text-sm font-medium mb-2 text-gray-900 dark:text-white">
              Work Email*
            </label>
            <input
              type="email"
              id="email"
              placeholder="Work Email"
              name="email"
              value={formData.email}
              onInput={e =>
                setFormData({...formData, email: e.currentTarget.value})
              }
              required
              class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors"
            />
          </div>
          <div>
            <label
              htmlFor="phone"
              class="block text-sm font-medium mb-2 text-gray-900 dark:text-white">
              Work Phone
            </label>
            <input
              type="tel"
              id="phone"
              placeholder="Work Phone"
              value={formData.phone}
              onInput={e =>
                setFormData({...formData, phone: e.currentTarget.value})
              }
              name="phone"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors"
            />
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="jobTitle"
              class="block text-sm font-medium mb-2 text-gray-900 dark:text-white">
              Job title*
            </label>
            <input
              type="text"
              id="jobTitle"
              placeholder="Job Title"
              name="jobTitle"
              value={formData.jobTitle}
              onInput={e =>
                setFormData({...formData, jobTitle: e.currentTarget.value})
              }
              required
              class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors"
            />
          </div>
          <div>
            <label
              htmlFor="companyName"
              class="block text-sm font-medium mb-2 text-gray-900 dark:text-white">
              Company Name*
            </label>
            <input
              type="text"
              id="companyName"
              placeholder="Company Name"
              name="companyName"
              value={formData.companyName}
              onInput={e =>
                setFormData({...formData, companyName: e.currentTarget.value})
              }
              required
              class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors"
            />
          </div>
        </div>

        {/* reCAPTCHA and Submit Button */}
        <div class="flex flex-col md:flex-row gap-4 items-center md:justify-between">
          <div
            class="g-recaptcha"
            data-sitekey={reCaptchaKeyV2}
            data-callback="onWhitePaperRecaptchaCallback"></div>

          <button
            type="submit"
            disabled={isSubmitting || !reCaptchaVerified}
            class="w-full md:w-80 px-8 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-semibold rounded-lg transition-colors duration-200 disabled:cursor-not-allowed cursor-pointer">
            {isSubmitting ? 'Submitting...' : 'Submit'}
          </button>
        </div>
      </form>
    </div>
  );
}

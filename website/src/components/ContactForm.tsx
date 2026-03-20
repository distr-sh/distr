import {useEffect, useState} from 'preact/hooks';

interface ContactRequest {
  firstName: string;
  lastName: string;
  email: string;
  phone: string;
  jobTitle: string;
  companyName: string;
  useCase: string;
}

interface ContactFormProps {
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

export default function ContactForm({
  formsServerBaseUrl,
  reCaptchaKeyV2,
}: ContactFormProps) {
  const [formData, setFormData] = useState<ContactRequest>({
    firstName: '',
    lastName: '',
    email: '',
    phone: '',
    jobTitle: '',
    companyName: '',
    useCase: '',
  });

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [reCaptchaVerified, setReCaptchaVerified] = useState(false);
  const [reCaptchaToken, setReCaptchaToken] = useState('');

  const handleChange = (
    e: Event & {currentTarget: HTMLInputElement | HTMLTextAreaElement},
  ) => {
    const {name, value} = e.currentTarget;
    setFormData(prev => ({...prev, [name]: value}));
  };

  // Load reCAPTCHA script and set up callback on component mount
  useEffect(() => {
    loadRecaptchaScript();

    // Set up global callback for reCAPTCHA
    (window as any).onContactRecaptchaCallback = (token: string) => {
      setReCaptchaVerified(true);
      setReCaptchaToken(token);
    };

    return () => {
      delete (window as any).onContactRecaptchaCallback;
    };
  }, []);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();

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
      const response = await fetch(`${formsServerBaseUrl}/contact`, {
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

      if (!response.ok) {
        throw new Error('Failed to submit form');
      }

      alert(
        'Thank you for reaching out. We will follow up as soon as possible.',
      );

      // Reset form
      setFormData({
        firstName: '',
        lastName: '',
        email: '',
        phone: '',
        jobTitle: '',
        companyName: '',
        useCase: '',
      });

      // Reset reCAPTCHA
      setReCaptchaVerified(false);
      setReCaptchaToken('');
      if ((window as any).grecaptcha) {
        (window as any).grecaptcha.reset();
      }
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : 'An error occurred';
      alert(errorMessage);
      console.error(errorMessage);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} class="flex flex-col gap-6">
      {/* First Name and Last Name */}
      <div class="grid md:grid-cols-2 gap-4">
        <div>
          <label
            for="firstName"
            class="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-200">
            First Name*
          </label>
          <input
            type="text"
            id="firstName"
            name="firstName"
            placeholder="First name"
            value={formData.firstName}
            onInput={handleChange}
            required
            class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:border-accent-600 dark:focus:border-accent-400 focus:ring-2 focus:ring-accent-600/20 dark:focus:ring-accent-400/20 focus:outline-none transition-colors"
          />
        </div>
        <div>
          <label
            for="lastName"
            class="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-200">
            Last Name*
          </label>
          <input
            type="text"
            id="lastName"
            name="lastName"
            placeholder="Last name"
            value={formData.lastName}
            onInput={handleChange}
            required
            class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:border-accent-600 dark:focus:border-accent-400 focus:ring-2 focus:ring-accent-600/20 dark:focus:ring-accent-400/20 focus:outline-none transition-colors"
          />
        </div>
      </div>

      {/* Email and Phone */}
      <div class="grid md:grid-cols-2 gap-4">
        <div>
          <label
            for="email"
            class="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-200">
            Work Email*
          </label>
          <input
            type="email"
            id="email"
            name="email"
            placeholder="Work Email"
            value={formData.email}
            onInput={handleChange}
            required
            class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:border-accent-600 dark:focus:border-accent-400 focus:ring-2 focus:ring-accent-600/20 dark:focus:ring-accent-400/20 focus:outline-none transition-colors"
          />
        </div>
        <div>
          <label
            for="phone"
            class="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-200">
            Work Phone
          </label>
          <input
            type="tel"
            id="phone"
            name="phone"
            placeholder="Work Phone"
            value={formData.phone}
            onInput={handleChange}
            class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:border-accent-600 dark:focus:border-accent-400 focus:ring-2 focus:ring-accent-600/20 dark:focus:ring-accent-400/20 focus:outline-none transition-colors"
          />
        </div>
      </div>

      {/* Job Title and Company Name */}
      <div class="grid md:grid-cols-2 gap-4">
        <div>
          <label
            for="jobTitle"
            class="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-200">
            Job title*
          </label>
          <input
            type="text"
            id="jobTitle"
            name="jobTitle"
            placeholder="Job Title"
            value={formData.jobTitle}
            onInput={handleChange}
            required
            class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:border-accent-600 dark:focus:border-accent-400 focus:ring-2 focus:ring-accent-600/20 dark:focus:ring-accent-400/20 focus:outline-none transition-colors"
          />
        </div>
        <div>
          <label
            for="companyName"
            class="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-200">
            Company Name*
          </label>
          <input
            type="text"
            id="companyName"
            name="companyName"
            placeholder="Company Name"
            value={formData.companyName}
            onInput={handleChange}
            required
            class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:border-accent-600 dark:focus:border-accent-400 focus:ring-2 focus:ring-accent-600/20 dark:focus:ring-accent-400/20 focus:outline-none transition-colors"
          />
        </div>
      </div>

      {/* Use Case */}
      <div>
        <label
          for="useCase"
          class="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-200">
          Use case*
        </label>
        <textarea
          id="useCase"
          name="useCase"
          placeholder="Tell us about your use case"
          value={formData.useCase}
          onInput={handleChange}
          required
          rows={5}
          class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:border-accent-600 dark:focus:border-accent-400 focus:ring-2 focus:ring-accent-600/20 dark:focus:ring-accent-400/20 focus:outline-none transition-colors resize-y"
        />
      </div>

      {/* reCAPTCHA and Submit Button */}
      <div class="flex flex-col md:flex-row gap-4 items-center md:justify-between">
        <div
          class="g-recaptcha"
          data-sitekey={reCaptchaKeyV2}
          data-callback="onContactRecaptchaCallback"></div>

        <button
          type="submit"
          disabled={isSubmitting || !reCaptchaVerified}
          class="w-full md:w-80 px-8 py-4 text-lg font-medium text-white bg-accent-600 hover:bg-accent-700 disabled:bg-gray-400 disabled:cursor-not-allowed rounded-lg transition-colors cursor-pointer">
          {isSubmitting ? 'Submitting...' : 'Submit'}
        </button>
      </div>
    </form>
  );
}

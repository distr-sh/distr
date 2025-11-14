import {useState} from 'preact/hooks';

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
}

function loadScript() {
  if (typeof window === 'undefined') {
    return;
  }

  const elementId = 'hs-script';
  if (document.getElementById(elementId) === null) {
    const script = document.createElement('script');
    script.type = 'text/javascript';
    script.src = 'https://js-eu1.hs-scripts.com/144345473.js';
    script.id = elementId;
    document.head.appendChild(script);
  }
}

export default function WhitePaperForm({
  formsServerBaseUrl,
}: WhitePaperFormProps) {
  const [formData, setFormData] = useState<WhitePaperRequest>({
    firstName: '',
    lastName: '',
    email: '',
    phone: '',
    jobTitle: '',
    companyName: '',
  });
  const [hubSpotLoaded, setHubSpotLoaded] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const loadHubSpot = () => {
    if (!hubSpotLoaded) {
      loadScript();
      setHubSpotLoaded(true);
    }
  };

  const handleSubmit = async (event: Event) => {
    event.preventDefault();
    setIsSubmitting(true);
    loadScript();

    try {
      const response = await fetch(`${formsServerBaseUrl}/white-paper`, {
        method: 'POST',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(formData),
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
              onInput={e => {
                setFormData({...formData, email: e.currentTarget.value});
                loadHubSpot();
              }}
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

        <button
          type="submit"
          disabled={isSubmitting}
          class="mt-2 px-8 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-semibold rounded-lg transition-colors duration-200 disabled:cursor-not-allowed">
          {isSubmitting ? 'Submitting...' : 'Submit'}
        </button>
      </form>
    </div>
  );
}

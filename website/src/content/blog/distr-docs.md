---
title: 'Distr Launch Week: Our docs are now built with Starlight instead of Docusaurus'
description: 'Comparing Docusaurus vs Distr incl. Design, SEO, DevEx, Extensibility and how we structured our technical documentation'
publishDate: 2025-05-23
lastUpdated: 2025-05-23
slug: 'distr-docs'
authors:
  - name: 'Philip Miglinci'
    role: 'Co-Founder'
    image: '/src/assets/blog/authors/pmig.jpg'
    linkedIn: https://www.linkedin.com/in/pmigat/
    gitHub: https://github.com/pmig
image: '/src/assets/blog/2025-05-23-distr-docs/distr-docs-thumbnail.png'
tags:
  - distr-launch-week
  - docs
  - docusaurus
  - astro
  - starlight
---

I am Philip—an engineer working at Glasskube. We are building an Open Source control plane for self-managed, BYOC, and on-prem deployments.
Everything you need to distribute applications to self-managed customers out of the box is called Distr
([`github.com/glasskube/distr`](https://github.com/glasskube/distr)).

We previously built all our technical documentation—including this website—with Docusaurus.
We generally liked using a static page generator and having all content fully Open Source on GitHub, so everyone can create a pull request
if they find a typo or want to improve our docs.
But for Distr, we decided to switch to Starlight.
In this blog post, we will compare Docusaurus and Starlight and explain why we made the switch.
Additionally, we will share how we structured our technical documentation and what we learned along the way.

<hr>

This post is actually the last piece of the Distr launch week.

Up until now, we launched:

- On Monday: [Tutorials for easy onboarding](/blog/distr-tutorials/)
- On Tuesday: [Docker Swarm Support](/blog/distr-docker-swarm/)
- On Wednesday: [Dashboard Unification with metrics](/blog/distr-dashboard/)
- On Thursday: [OCI Artifact Registry is GA](/blog/distr-registry/)

Read all about the Distr launch week here: [`#distr-launch-week`](/blog/).

Coming back to the original topic:
You can check out the source code of this exact website at [`github.com/glasskube/glasskube.dev`](https://github.com/glasskube/glasskube.dev),
and you will see, it is built with Docusaurus.

Although Docusaurus did its job quite well, we decided to use [Starlight](https://starlight.astro.build/)
which is built on top of [Astro](https://astro.build/) for our Distr docs.
After working with both frameworks, I am able to summarize the differences and point out the biggest distinctions.

There are many popular open source frameworks for building great documentation and even more static site renderers (SSR),
which can be customized to serve documentation.

To some extent, choosing Starlight was a gut decision, and we are only able to compare Astro to Docusaurus.
Another alternative worth mentioning is MkDocs ([mkdocs.org](https://www.mkdocs.org/)).
[Material for MkDocs](https://squidfunk.github.io/mkdocs-material/) in particular looks really stunning.

## Docusaurus vs. Starlight - What to look at?

When evaluating a static site renderer for our documentation, I'm looking for something that fulfills the following criteria:

- **Design**: A nice documentation framework looks sleek and modern by default but can still be easily customized to look unique.
- **SEO**: All required SEO features (description, sitemaps, page structures) should be available and built into the framework.
- **DevEx**: Developer experience should be great.
  Build times and reloads should be fast, and there shouldn't be an endless number of third-party dependencies to get started.
- **Extensibility:** It should be possible to also create marketing pages with the same framework.

Another factor to look at is obviously maturity and age. Although Docusaurus was first released in 2017 by Joel Marcey,
see his blog post about [Introducing Docusaurus](https://docusaurus.io/blog/2017/12/14/introducing-docusaurus), it is still actively maintained.

[Starlight v0.0.1](https://github.com/withastro/starlight/releases/tag/%40astrojs%2Fstarlight%400.0.19) was first released in 2023, so 5 years later.

### Docusaurus vs. Starlight: Design

Docusaurus and Starlight have made quite different decisions regarding design.

Docusaurus uses Infima ([infima.dev](https://infima.dev/)) at its core.
It is actually pretty tightly coupled, so if you want to style specific pages or use CSS classes to style some components, you need to get familiar with Infima.
Sadly, Infima isn't very mature yet and it doesn't provide primitives for a lot of components and possibilities that a framework like Bootstrap offers.

Also, integrating Bootstrap or Tailwind is not really possible.

Starlight and Astro, on the other hand, don't come with a specific CSS framework built in, but provide a one-line command with their CLI to integrate Tailwind.

Read more about [Starlight and Tailwind](https://starlight.astro.build/guides/css-and-tailwind/).

:::note
It took a while until Starlight fully supported Tailwind v4.
But once [#2862](https://github.com/withastro/starlight/issues/2862) was completed,
upgrading Starlight from Tailwind v3 to v4 worked very well and was pretty well documented.
:::

### Docusaurus vs. Starlight: SEO

Both frameworks provide all the necessary features to configure HTML tags like `<description>` out of the box.
You can read the [SEO guide for Docusaurus](https://docusaurus.io/docs/seo) or check out all the
[SEO Integrations for Astro](https://astro.build/integrations/?search=&categories%5B%5D=performance%2Bseo)
which come preconfigured with Starlight.

As both frameworks support static rendering at build time, page loading is lightning fast and is only slowed down
by external libraries that are integrated.

| Feature                          | Docusaurus                  | Starlight |
| -------------------------------- | --------------------------- | --------- |
| Sitemap generation               | ✅                          | ✅        |
| Metadata & Open Graph            | ✅                          | ✅        |
| Broken Link Checker              | ✅                          | ✅        |
| Image Optimization & compression | ⚠️ (hard to get it working) | ✅        |

It should be noted that all features for Starlight are only available via plugins and not part of the core Astro framework.

:::tip
I strongly recommend to always run your website through Google's [PageSpeed Insights Test](https://pagespeed.web.dev/)
:::

### Docusaurus vs. Starlight: Developer experience

:::info
Both Docusaurus and Starlight are built on top of existing web frameworks.
While Docusaurus is built upon React, Starlight is based upon Astro.
Astro itself can be used with multiple different frameworks, including React, Vue, and Svelte.
:::

A framework for writing documentation should not hold you back from writing documentation quickly:

- It should not be a lot of effort to keep up to date.
- It should be fast, blazing fast.
- It should have useful integrations, documentation, and an active community.

#### Maintenance work

Docusaurus' hard dependency on React introduces a natural update hurdle.
Although React, Docusaurus, and Infima are all Open Source projects from Meta (formerly Facebook), they are developed by different teams.
Not only does the Docusaurus team and you need to test compatibility with the latest React version, but also all plugins need to ensure compatibility.
Only after the core team and all plugins support the latest version of React can you safely upgrade your Docusaurus version.

Starlight, on the other hand, only depends on Astro, which is the same team building Starlight. This doesn't seem to require the same amount of effort to keep up to date.
It also has fewer moving parts.

#### Speed

To be completely honest: The reason I looked for another documentation framework was that I was fed up with Docusaurus taking forever to start the initial development server.
With the latest Node version (24), Docusaurus version (3.7), and React version (19), it took a little over 5 seconds to start the development server.
Although this number doesn't seem to be that high, it was a lot higher with previous versions.
I remember waiting over 20 seconds for the development server to start, which _really_ frustrated me.

Starlight, on the other hand, starts the development server in less than half the time Docusaurus takes.
The biggest difference is that the Astro compiler [`github.com/withastro/compiler`](https://github.com/withastro/compiler)
is written in Go instead of using JavaScript to build the website bundle.

#### Integrations

The only integration that is still missing and not available for Starlight is a Mermaid integration.
Although Mermaid support ([#1259](https://github.com/withastro/starlight/discussions/1259)) is the second most requested feature after multiple sidebars
([#959](https://github.com/withastro/starlight/discussions/959)), there is no official statement if it is on the roadmap.
Also, no community plugin has yet been created.
Currently, there is only a workaround with rendering Mermaid charts with Playwright.

Find all Starlight plugins in the [Starlight plugin directory](https://starlight.astro.build/resources/plugins/) and all
Docusaurus integrations in the [Docusaurus plugin directory](https://docusaurus.io/docs/api/plugins).

#### Feeling

Starlight and Astro also felt more fun to use.
Their CLI has nice animations, everything just looks a little bit more modern with all the fancy animations in the terminal.

### Docusaurus vs. Starlight: Extensibility—Adding Marketing pages

When I tried to create marketing pages with Starlight in addition to the technical documentation, I nearly gave up.
Coming from the Docusaurus world, this wasn't an issue as the starter template comes with a front page and a blog out of the box.
You can even create multiple documentations on different paths.
We used `/docs`, for example.

Starlight, on the other hand, is _only_ built for documentation and not for marketing pages.
It even took an ugly hack to make sure the default path is `/docs` and not `/`.

Please don't look at this custom script configured in our [`astro.config.mjs`](https://github.com/glasskube/distr.sh/blob/main/astro.config.mjs#L23)
we need to execute on every page to make sure that the redirection works properly:

```json
{
  "tag": "script",
  "content": "`window.addEventListener('load', () => document.querySelector('.site-title').href += 'docs/')"
}
```

We haven't created any other marketing pages yet, but I am afraid that they will look quite a bit different from the Starlight documentation.

## Writing a technical documentation—What is important?

When writing technical documentation, there are a few things to keep in mind.
We will go over how we structured the content and what effects we observed on the viewed pages.

### Technical documentation structure

Distr isn't built for just one type of scenario.

So rather than simply documenting features, we took inspiration from other project docs we really like,
such as [Flux](https://fluxcd.io/flux/) and [Docker](https://docs.docker.com/) and similarly structured our content.
To first introduce core conceptual aspects relating to self-managed deployments and Distr more generally,
and from there, we connect those foundational concepts directly to practical use cases.

New users typically start their journey in our **Getting Started** section.
Here, we address questions such as "What exactly is Distr?", "Who is it built for?" and "What can Distr do?"

<div
  className="app-frame mac dark borderless shadow--tl"
  data-url="distr.sh/docs/getting-started/what-is-distr/">
  ![Screenshot of the Distr
  docs](/src/assets/blog/2025-05-23-distr-docs/9e0436c7-839a-40b5-8e0a-073cd80d6adf.png)
</div>

Afterward, users are prompted to move on to the **Use Cases** section.
Ideally, this is the stage where users identify scenarios similar to their own deployment challenges.

<div
  className="app-frame mac dark borderless shadow--tl"
  data-url="distr.sh/docs/use-cases/fully-self-managed/">
  ![Screenshot of the Distr
  docs](/src/assets/blog/2025-05-23-distr-docs/3744fe91-c12d-470e-affe-537d59748ee3.png)
</div>

Only then do we guide the user to the detailed technical insights and precise implementation instructions found in the **Guides and Product** sections.

<div
  className="app-frame mac dark borderless shadow--tl"
  data-url="distr.sh/docs/product/registry/">
  ![Screenshot of the Distr
  docs](/src/assets/blog/2025-05-23-distr-docs/0f1c1850-4be4-4421-a6fa-ae31fc1adad6.png)
</div>

Additionally, we offer complementary documentation for self-hosting Distr itself.
Oh, and in case you were wondering: Yes, you can actually use Distr to self-host Distr!
We also provide detailed resources on integrations, including how to leverage our API and SDK, as well as guides covering automation and authentication topics.

### Focusing on the most viewed pages

Planning how your documentation should ideally be consumed is one thing, but understanding how it actually plays out in reality is another.
By leveraging PostHog to monitor our documentation's usage metrics, we've observed something probably obvious.

![](/src/assets/blog/2025-05-23-distr-docs/79865ecd-32a3-41b5-9c2e-d3eb3c4d5bea.png)

People read the documentation often like a book, starting from the beginning and working their way through.
So if there is any information you want them to read, make sure to put it at the beginning of the documentation.

### Writing style for technical documentation

Let's be honest, most people don't spend their weekends curled up with documentation and a cup of warm cocoa.
Good documentation doesn't need poetic or elaborate prose.
It just has to be clear, concise, and intuitively skimmable.
When we write docs, we really try to remove unnecessary fluff and filler words and not be overly verbose.

Every sentence should be in service to the user's ability to quickly understand what your product does and, most importantly, how it solves the user's specific problems.
Fighting to retain attention is tough for any medium, but documentation writers have it the hardest since this is the prevailing sentiment they are up against.

![Screenshot of the Distr doc](/src/assets/blog/2025-05-23-distr-docs/cbdd8528-bf64-433d-aaea-7e7d5a7b6c37.webp)

Making an effort to be clear, concise, and highly reliable has its payoffs though, since it doesn't just help the human readers,
but it also ensures that docs are structured in such a way that AI models can reliably interpret and communicate accurate information on your behalf.

It's also true that visuals can often communicate more clearly than words alone.
That's why we pair concise explanations with detailed, annotated screenshots and diagrams where needed.

<div
  className="app-frame mac dark borderless shadow--tl"
  data-url="distr.sh/docs/product/vendor-portal/">
  ![](/src/assets/blog/2025-05-23-distr-docs/1cf9c44f-50dd-462c-881b-f7bd70f88473.png)
</div>

## Conclusion

Switching to a new technology always comes with a learning curve, but this time the learning curve wasn't that steep.

We are more than happy with the decision to switch to Starlight and Astro.
The developer experience is much better, and the documentation looks way more modern out of the box.
Although it was hard to accept that it is harder to build your marketing website and blog also with Starlight, and Mermaid integration is still missing,
we are happy with the decision to use Starlight for our technical documentation and will also use it in our next projects.

We would like to get your opinion.
Which of the two documentation frameworks do you prefer?

- [`glasskube.dev`](https://glasskube.dev) built with Docusaurus
- [`distr.sh`](https://distr.sh/docs/getting-started/what-is-distr/) built with Starlight

P.S.: Read all our announcements of this week's launch week here: [`#distr-launch-week`](/blog/)

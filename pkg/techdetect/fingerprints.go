package techdetect

// defaultFingerprints returns a curated set of technology fingerprints
// covering common web technologies: JS frameworks, CMS, web servers, etc.
// Patterns are regex-based for flexibility.
func defaultFingerprints() []Fingerprint {
	return []Fingerprint{
		// ===== JavaScript Frameworks & Libraries =====
		{
			Name:     "jQuery",
			Category: "js",
			Script:   []string{`jquery[.\-]?(\d+\.\d+[\.\d]*)?.*\.js`},
			HTML:     []string{`jquery`},
			Version:  `jquery[.\-]?(\d+\.\d+[\.\d]*)`,
			Priority: 10,
		},
		{
			Name:     "React",
			Category: "js",
			HTML:     []string{`data-reactroot`, `_reactRootContainer`, `__NEXT_DATA__`},
			Script:   []string{`react.*\.js`, `react-dom.*\.js`},
			Priority: 10,
		},
		{
			Name:     "Vue.js",
			Category: "js",
			HTML:     []string{`data-v-[a-f0-9]+`, `__vue_app__`, `v-cloak`},
			Script:   []string{`vue.*\.js`},
			Meta:     map[string]string{"": ""},
			Priority: 10,
		},
		{
			Name:     "Angular",
			Category: "js",
			HTML:     []string{`ng-app`, `ng-controller`, `ng-model`, `_nghost`, `_ngcontent`},
			Script:   []string{`angular.*\.js`, `@angular`},
			Priority: 10,
		},
		{
			Name:     "Next.js",
			Category: "js",
			HTML:     []string{`__NEXT_DATA__`, `__next`},
			Headers:  map[string]string{"x-nextjs-cache": "."},
			Priority: 10,
		},
		{
			Name:     "Nuxt.js",
			Category: "js",
			HTML:     []string{`__NUXT__`, `data-nuxt`, `nuxt`},
			Priority: 10,
		},
		{
			Name:     "Svelte",
			Category: "js",
			HTML:     []string{`class="svelte-[a-z0-9]+"`},
			Priority: 10,
		},
		{
			Name:     "Bootstrap",
			Category: "js",
			HTML:     []string{`bootstrap\.min\.css`, `bootstrap\.css`},
			Script:   []string{`bootstrap.*\.js`},
			Version:  `bootstrap[./-](\d+\.\d+[\.\d]*)`,
			Priority: 8,
		},
		{
			Name:     "Tailwind CSS",
			Category: "js",
			HTML:     []string{`tailwind\.css`, `class="[^"]*\b(flex|grid|bg-|text-|p-|m-|w-|h-|rounded|shadow|border)[^"]*\b"`},
			Priority: 7,
		},
		{
			Name:     "Lodash",
			Category: "js",
			Script:   []string{`lodash.*\.js`},
			Priority: 5,
		},
		{
			Name:     "Chart.js",
			Category: "js",
			Script:   []string{`chart\.js`, `chartjs`},
			Priority: 5,
		},
		{
			Name:     "D3.js",
			Category: "js",
			Script:   []string{`d3\.js`, `d3\.min\.js`},
			Priority: 5,
		},

		// ===== CMS & Blog Platforms =====
		{
			Name:     "WordPress",
			Category: "cms",
			HTML:     []string{`wp-content`, `wp-includes`, `wordpress`},
			Headers:  map[string]string{"link": `rel="https://api\.w\.org/"`},
			Meta:     map[string]string{"generator": `WordPress`},
			Script:   []string{`wp-includes`, `wp-content`},
			Version:  `WordPress\s+(\d+\.\d+[\.\d]*)`,
			Priority: 15,
		},
		{
			Name:     "Drupal",
			Category: "cms",
			HTML:     []string{`Drupal`, `drupal\.js`, `sites/default/files`},
			Meta:     map[string]string{"generator": `Drupal`},
			Headers:  map[string]string{"x-drupal-cache": "."},
			Priority: 15,
		},
		{
			Name:     "Joomla",
			Category: "cms",
			HTML:     []string{`/media/jui/`, `/media/system/`},
			Meta:     map[string]string{"generator": `Joomla`},
			Priority: 15,
		},
		{
			Name:     "Hugo",
			Category: "cms",
			HTML:     []string{`hugo-`, `Hugo`},
			Meta:     map[string]string{"generator": `Hugo`},
			Version:  `Hugo\s+(\d+\.\d+[\.\d]*)`,
			Priority: 10,
		},
		{
			Name:     "Hexo",
			Category: "cms",
			Meta:     map[string]string{"generator": `Hexo`},
			Version:  `Hexo\s+(\d+\.\d+[\.\d]*)`,
			Priority: 10,
		},
		{
			Name:     "Ghost",
			Category: "cms",
			Meta:     map[string]string{"generator": `Ghost`},
			Version:  `Ghost\s+(\d+\.\d+[\.\d]*)`,
			Priority: 10,
		},
		{
			Name:     "Typecho",
			Category: "cms",
			Meta:     map[string]string{"generator": `Typecho`},
			Version:  `Typecho\s+(\d+\.\d+[\.\d]*)`,
			Priority: 10,
		},

		// ===== Web Servers =====
		{
			Name:     "Nginx",
			Category: "webserver",
			Headers:  map[string]string{"server": `nginx`},
			Version:  `nginx/(\d+\.\d+[\.\d]*)`,
			Priority: 12,
		},
		{
			Name:     "Apache",
			Category: "webserver",
			Headers:  map[string]string{"server": `(?:Apache|httpd)`},
			Version:  `Apache/(\d+\.\d+[\.\d]*)`,
			Priority: 12,
		},
		{
			Name:     "Apache Tomcat",
			Category: "webserver",
			Headers:  map[string]string{"server": `(?:Apache-Coyote|Tomcat)`},
			Version:  `Apache[- ]Tomcat/(\d+\.\d+[\.\d]*)`,
			Priority: 12,
		},
		{
			Name:     "IIS",
			Category: "webserver",
			Headers:  map[string]string{"server": `Microsoft-IIS`},
			Version:  `Microsoft-IIS/(\d+\.\d+)`,
			Priority: 12,
		},
		{
			Name:     "Caddy",
			Category: "webserver",
			Headers:  map[string]string{"server": `Caddy`},
			Version:  `Caddy(?:/(\d+\.\d+[\.\d]*))?`,
			Priority: 12,
		},
		{
			Name:     "LiteSpeed",
			Category: "webserver",
			Headers:  map[string]string{"server": `LiteSpeed`},
			Priority: 12,
		},
		{
			Name:     "OpenResty",
			Category: "webserver",
			Headers:  map[string]string{"server": `openresty`},
			Version:  `openresty/(\d+\.\d+[\.\d]*)`,
			Priority: 12,
		},

		// ===== Programming Languages (detected via headers) =====
		{
			Name:     "PHP",
			Category: "language",
			Headers:  map[string]string{"x-powered-by": `PHP`},
			HTML:     []string{`\.php`},
			Version:  `PHP/(\d+\.\d+[\.\d]*)`,
			Priority: 10,
		},
		{
			Name:     "Express",
			Category: "language",
			Headers:  map[string]string{"x-powered-by": `Express`},
			Version:  `Express(?:/(\d+\.\d+[\.\d]*))?`,
			Priority: 10,
		},
		{
			Name:     "ASP.NET",
			Category: "language",
			Headers:  map[string]string{"x-aspnet-version": `.`},
			HTML:     []string{`__VIEWSTATE`, `asp\.net`},
			Priority: 10,
		},

		// ===== CDNs & Cloud =====
		{
			Name:     "Cloudflare",
			Category: "cdn",
			Headers:  map[string]string{"cf-ray": ".", "server": `cloudflare`},
			Priority: 12,
		},
		{
			Name:     "Vercel",
			Category: "cdn",
			Headers:  map[string]string{"x-vercel-id": ".", "x-vercel-cache": "."},
			Priority: 10,
		},
		{
			Name:     "Netlify",
			Category: "cdn",
			Headers:  map[string]string{"x-nf-request-id": "."},
			Priority: 10,
		},
		{
			Name:     "Akamai",
			Category: "cdn",
			Headers:  map[string]string{"x-akamai-transformed": ".", "x-cache": `Akamai`},
			Priority: 10,
		},

		// ===== Analytics & Tracking =====
		{
			Name:     "Google Analytics",
			Category: "analytics",
			HTML:     []string{`google-analytics\.com`, `gtag`, `gtagjs`, `UA-\d+-\d+`, `G-[A-Z0-9]+`},
			Script:   []string{`google-analytics`, `gtag`},
			Priority: 8,
		},
		{
			Name:     "Google Tag Manager",
			Category: "analytics",
			HTML:     []string{`googletagmanager\.com`, `GTM-[A-Z0-9]+`},
			Priority: 8,
		},
		{
			Name:     "Baidu Analytics",
			Category: "analytics",
			HTML:     []string{`hm\.baidu\.com`, `hmt\.js`},
			Script:   []string{`hm\.baidu\.com`},
			Priority: 8,
		},
		{
			Name:     "Matomo",
			Category: "analytics",
			HTML:     []string{`matomo\.js`, `piwik\.js`},
			Script:   []string{`matomo`, `piwik`},
			Priority: 8,
		},

		// ===== Security & Auth =====
		{
			Name:     "Cloudflare Rocket Loader",
			Category: "security",
			HTML:     []string{`rocket-loader\.min\.js`, `data-cf-beacon`},
			Priority: 5,
		},

		// ===== E-commerce =====
		{
			Name:     "Shopify",
			Category: "ecommerce",
			HTML:     []string{`shopify`, `Shopify\.theme`},
			Headers:  map[string]string{"x-shopid": "."},
			Priority: 10,
		},
		{
			Name:     "Magento",
			Category: "ecommerce",
			HTML:     []string{`mage/`, `Magento_`, `skin/frontend`},
			Meta:     map[string]string{"generator": `Magento`},
			Priority: 10,
		},

		// ===== Build Tools & Dev =====
		{
			Name:     "Webpack",
			Category: "build",
			HTML:     []string{`webpack`, `__webpack_require__`},
			Script:   []string{`webpack`},
			Priority: 5,
		},
		{
			Name:     "Vite",
			Category: "build",
			HTML:     []string{`vite/modulepreload-polyfill`, `@vite/client`},
			Script:   []string{`@vite`},
			Priority: 5,
		},

		// ===== Misc =====
		{
			Name:     "Font Awesome",
			Category: "misc",
			HTML:     []string{`font-awesome`, `fontawesome`, `fa[sblr]? fa-`},
			Priority: 3,
		},
		{
			Name:     "reCAPTCHA",
			Category: "security",
			HTML:     []string{`recaptcha`, `g-recaptcha`, `google\.com/recaptcha`},
			Script:   []string{`recaptcha`},
			Priority: 5,
		},
		{
			Name:     "Grafana",
			Category: "monitoring",
			HTML:     []string{`grafana-app`, `Grafana`},
			Priority: 8,
		},
		{
			Name:     "Jenkins",
			Category: "ci",
			HTML:     []string{`jenkins`, `Jenkins-Crumb`},
			Headers:  map[string]string{"x-jenkins": "."},
			Priority: 10,
		},
		{
			Name:     "GitLab",
			Category: "ci",
			HTML:     []string{`gitlab`, `GitLab`},
			Headers:  map[string]string{"x-gitlab": "."},
			Priority: 10,
		},
	}
}

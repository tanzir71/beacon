(() => {
  "use strict";
  const page = document.documentElement;
  const root = new URL(page.dataset.root, location.href);
  const valid = value => value === "en" || value === "bn";
  const get = key => { try { return localStorage.getItem(key); } catch { return null; } };
  const set = (key, value) => { try { localStorage.setItem(key, value); } catch { /* Preferences are optional. */ } };
  const explicit = new URLSearchParams(location.search).get("lang");
  if (valid(explicit)) set("beacon.language", explicit);
  let choice = valid(explicit) ? explicit : null;
  if (!choice && page.dataset.landingRoot === "true") {
    const stored = get("beacon.language");
    choice = valid(stored) ? stored : (navigator.languages || [navigator.language]).some(language => /^bn(?:-|$)/i.test(language)) ? "bn" : "en";
  }
  if (choice && choice !== page.dataset.locale) {
    location.replace(new URL(`${choice}/`, root).href);
    return;
  }
  document.querySelectorAll("[data-language]").forEach(link => {
    link.addEventListener("click", () => set("beacon.language", link.dataset.language));
  });
  const savedTheme = get("beacon.theme");
  if (savedTheme === "day" || savedTheme === "night") page.dataset.theme = savedTheme;
  document.querySelector("[data-theme-toggle]").addEventListener("click", () => {
    const day = page.dataset.theme ? page.dataset.theme === "day" : matchMedia("(prefers-color-scheme: light)").matches;
    page.dataset.theme = day ? "night" : "day";
    set("beacon.theme", page.dataset.theme);
  });
  document.querySelectorAll("[data-reach]").forEach(button => {
    button.addEventListener("click", () => {
      document.querySelectorAll("[data-reach]").forEach(step => step.setAttribute("aria-pressed", String(step === button)));
    });
  });
  const share = document.querySelector("[data-share]");
  share.addEventListener("click", async () => {
    const url = new URL(`${page.dataset.locale}/`, root).href;
    try {
      if (navigator.share) await navigator.share({ title: "Blackout Beacon", url });
      else {
        await navigator.clipboard.writeText(url);
        document.querySelector("[data-share-result]").textContent = share.dataset.copied;
      }
    } catch (error) {
      if (error.name !== "AbortError") location.href = `mailto:?subject=Blackout%20Beacon&body=${encodeURIComponent(url)}`;
    }
  });
})();

"use client";

import { LandingHeader } from "./landing-header";
import { LandingHero } from "./landing-hero";
import { FeaturesSection } from "./features-section";
import { HowItWorksSection } from "./how-it-works-section";
import { ContactSection } from "./contact-section";
import { OpenSourceSection } from "./open-source-section";
import { FAQSection } from "./faq-section";
import { LandingFooter } from "./landing-footer";

export function MulticaLanding() {
  return (
    <>
      <div className="relative">
        <LandingHeader />
        <LandingHero />
      </div>
      <FeaturesSection />
      <HowItWorksSection />
      <ContactSection />
      <OpenSourceSection />
      <FAQSection />
      <LandingFooter />
    </>
  );
}

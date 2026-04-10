import type { Metadata } from "next";
import { AboutPageClient } from "@/features/landing/components/about-page-client";

export const metadata: Metadata = {
  title: "О платформе",
  description:
    "Что такое Multica и зачем командам нужен управляемый слой для работы с coding agents.",
  openGraph: {
    title: "О Multica",
    description: "Идея и подход Multica к работе людей и AI-агентов.",
    url: "/about",
  },
  alternates: {
    canonical: "/about",
  },
};

export default function AboutPage() {
  return <AboutPageClient />;
}

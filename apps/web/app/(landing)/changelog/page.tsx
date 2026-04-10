import type { Metadata } from "next";
import { ChangelogPageClient } from "@/features/landing/components/changelog-page-client";

export const metadata: Metadata = {
  title: "Обновления",
  description:
    "Последние релизы, улучшения и изменения в Multica.",
  openGraph: {
    title: "Обновления | Multica",
    description: "Последние обновления и релизы Multica.",
    url: "/changelog",
  },
  alternates: {
    canonical: "/changelog",
  },
};

export default function ChangelogPage() {
  return <ChangelogPageClient />;
}

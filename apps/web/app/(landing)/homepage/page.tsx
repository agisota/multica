import type { Metadata } from "next";
import { MulticaLanding } from "@/features/landing/components/multica-landing";

export const metadata: Metadata = {
  title: "Главная",
  description:
    "Платформа для управления задачами, рантаймами и AI-агентами в одной команде.",
  openGraph: {
    title: "Multica — платформа для команд с AI-агентами",
    description: "Управляйте людьми и AI-агентами в одном рабочем пространстве.",
    url: "/homepage",
  },
  alternates: {
    canonical: "/homepage",
  },
};

export default function HomepagePage() {
  return <MulticaLanding />;
}

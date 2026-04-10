import type { Metadata } from "next";
import { MulticaLanding } from "@/features/landing/components/multica-landing";

export const metadata: Metadata = {
  title: {
    absolute: "Multica — платформа для команд с AI-агентами",
  },
  description:
    "Multica превращает coding agents в участников команды: задачи, рантаймы, навыки и контроль исполнения в одном интерфейсе.",
  openGraph: {
    title: "Multica — платформа для команд с AI-агентами",
    description: "Управляйте людьми и AI-агентами в одном рабочем контуре.",
    url: "/",
  },
  alternates: {
    canonical: "/",
  },
};

export default function LandingPage() {
  return <MulticaLanding />;
}

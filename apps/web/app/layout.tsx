import type { Metadata, Viewport } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { ThemeProvider } from "@/components/theme-provider";
import { Toaster } from "@multica/ui/components/ui/sonner";
import { cn } from "@multica/ui/lib/utils";
import { QueryProvider } from "@multica/core/provider";
import { AuthInitializer } from "@/features/auth";
import { WebWSProvider } from "@/platform/ws-provider";
import { WebNavigationProvider } from "@/platform/navigation";
import { LocaleSync } from "@/components/locale-sync";
import "./globals.css";

const geist = Geist({ subsets: ["latin"], variable: "--font-sans" });
const geistMono = Geist_Mono({ subsets: ["latin"], variable: "--font-mono" });

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#ffffff" },
    { media: "(prefers-color-scheme: dark)", color: "#05070b" },
  ],
};

export const metadata: Metadata = {
  metadataBase: new URL("https://multica.zed.md"),
  title: {
    default: "Multica — платформа для команд с AI-агентами",
    template: "%s | Multica",
  },
  description:
    "Multica превращает coding agents в участников команды: задачи, рантаймы, навыки и контроль исполнения в одном интерфейсе.",
  icons: {
    icon: [{ url: "/favicon.svg", type: "image/svg+xml" }],
    shortcut: ["/favicon.svg"],
  },
  openGraph: {
    type: "website",
    siteName: "Multica",
    locale: "ru_RU",
  },
  alternates: {
    canonical: "/",
  },
  robots: {
    index: true,
    follow: true,
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="ru"
      suppressHydrationWarning
      className={cn("antialiased font-sans h-full", geist.variable, geistMono.variable)}
    >
      <body className="h-full overflow-hidden">
        <LocaleSync />
        <ThemeProvider>
          <QueryProvider showDevtools={process.env.NEXT_PUBLIC_DEVTOOLS === "true"}>
            <WebNavigationProvider>
              <AuthInitializer>
                <WebWSProvider>{children}</WebWSProvider>
              </AuthInitializer>
            </WebNavigationProvider>
            <Toaster />
          </QueryProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}

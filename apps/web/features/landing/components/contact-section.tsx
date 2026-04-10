"use client";

import Link from "next/link";
import { useLocale } from "../i18n";

export function ContactSection() {
  const { t } = useLocale();

  return (
    <section id="contact" className="bg-[#f4efe6] text-[#0a0d12]">
      <div className="mx-auto grid max-w-[1320px] gap-12 px-4 py-24 sm:px-6 sm:py-32 lg:grid-cols-[0.95fr_1.05fr] lg:gap-20 lg:px-8 lg:py-40">
        <div>
          <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[#0a0d12]/40">
            {t.contact.label}
          </p>
          <h2 className="mt-4 font-[family-name:var(--font-serif)] text-[2.6rem] leading-[1.05] tracking-[-0.03em] sm:text-[3.4rem] lg:text-[4.2rem]">
            {t.contact.title}
          </h2>
          <p className="mt-6 max-w-[480px] text-[15px] leading-7 text-[#0a0d12]/60 sm:text-[16px]">
            {t.contact.description}
          </p>
        </div>
        <div className="rounded-[24px] border border-[#0a0d12]/10 bg-white p-6 shadow-[0_24px_80px_rgba(10,13,18,0.08)] sm:p-8">
          <div className="max-w-[520px]">
            <p className="text-sm font-medium text-[#0a0d12]/54">
              Канал связи с лендинга временно упрощён, чтобы страница стабильно загружалась в dev.
            </p>
            <p className="mt-4 text-[15px] leading-7 text-[#0a0d12]/70">
              Если хотите обсудить внедрение Multica, перейдите в продукт и запустите рабочий контур команды. После этого можно использовать обычный login-flow и внутренние рабочие процессы вместо нестабильной dev-формы.
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <Link
                href="/login"
                className="inline-flex h-12 items-center justify-center rounded-xl bg-[#0a0d12] px-5 text-[14px] font-semibold text-white"
              >
                Открыть вход
              </Link>
              <Link
                href="/about"
                className="inline-flex h-12 items-center justify-center rounded-xl border border-[#0a0d12]/12 px-5 text-[14px] font-semibold text-[#0a0d12]"
              >
                Узнать о платформе
              </Link>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

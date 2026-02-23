import type { Metadata } from "next";
import { Geist } from "next/font/google";
import "./globals.css";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AgentsProvider } from "@/components/AgentsProvider";
import { ThemeProvider } from "@/components/ThemeProvider";
import { Toaster } from "@/components/ui/sonner";
import { AppInitializer } from "@/components/AppInitializer";
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/sidebars/AppSidebar";
import { NamespaceProvider } from "@/contexts/NamespaceContext";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "kagent.dev",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <TooltipProvider>
      <AgentsProvider>
        <html lang="en" suppressHydrationWarning>
          <body className={`${geistSans.className} h-screen overflow-hidden`}>
            <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
              <NamespaceProvider>
                <SidebarProvider>
                  <AppSidebar />
                  <SidebarInset>
                    <AppInitializer>
                      <main className="flex-1 overflow-y-auto h-full">{children}</main>
                    </AppInitializer>
                  </SidebarInset>
                </SidebarProvider>
              </NamespaceProvider>
              <Toaster richColors />
            </ThemeProvider>
          </body>
        </html>
      </AgentsProvider>
    </TooltipProvider>
  );
}

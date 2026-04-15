import { CheckCircle2Icon, Sparkles, Zap, Rocket, Zap as ViteIcon, Atom, BookOpen, Palette, Layers, Beaker, Map, Database } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import BlurText from "../components/BlurText";
import GradientText from "../components/GradientText";
import FadeContent from "../components/FadeContent";
import ShinyText from "../components/ShinyText";

export function HomePage() {
  const techStack = [
    { icon: ViteIcon, name: "Vite", description: "Next Generation Frontend Tooling" },
    { icon: Atom, name: "React 19", description: "The library for web and native UIs" },
    { icon: BookOpen, name: "TypeScript", description: "JavaScript with syntax for types" },
    { icon: Palette, name: "Tailwind v4", description: "Utility-first CSS framework" },
    { icon: Layers, name: "shadcn/ui", description: "Beautifully designed components" },
    { icon: Beaker, name: "Vitest", description: "Blazing fast unit test framework" },
    { icon: Map, name: "React Router v7", description: "Declarative routing for React" },
    { icon: Database, name: "Zustand", description: "Bear necessities for state management" },
  ];

  const commands = [
    { cmd: "make install", desc: "Install dependencies" },
    { cmd: "make dev", desc: "Start dev server" },
    { cmd: "make test", desc: "Run tests once" },
    { cmd: "make test-watch", desc: "Run tests in watch mode" },
    { cmd: "make build", desc: "Production build" },
    { cmd: "make build-analyze", desc: "Build with bundle analyzer" },
    { cmd: "make lint", desc: "Check code with ESLint" },
    { cmd: "make lint-fix", desc: "Fix ESLint issues" },
    { cmd: "make format", desc: "Format code with Prettier" },
    { cmd: "make type-check", desc: "TypeScript type checking" },
  ];

  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-12 px-6 py-16">
        {/* Hero Section */}
        <FadeContent>
          <header className="flex flex-col items-center gap-6 text-center">
            <Badge className="gap-2 px-4 py-2 bg-gradient-to-r from-purple-500 via-pink-500 to-orange-500 text-white hover:from-purple-600 hover:via-pink-600 hover:to-orange-600">
              <Sparkles className="h-4 w-4 text-white" />
              <ShinyText text="Modern React Template" />
            </Badge>
            
            <div className="flex flex-col gap-4">
              <BlurText
                text="Build Fast, Ship Faster"
                className="text-5xl font-bold tracking-tight md:text-7xl"
                delay={50}
              />
              
              <GradientText
                className="text-xl text-muted-foreground md:text-2xl"
                colors={["#a855f7", "#ec4899", "#f97316"]}
              >
                A production-ready React template with TypeScript, Tailwind CSS, and shadcn/ui
              </GradientText>
            </div>

            <div className="flex flex-wrap items-center justify-center gap-3 pt-4">
              <Badge variant="outline" className="gap-1.5">
                <Zap className="h-3.5 w-3.5" />
                Lightning Fast
              </Badge>
              <Badge variant="outline" className="gap-1.5">
                <Rocket className="h-3.5 w-3.5" />
                Production Ready
              </Badge>
              <Badge variant="outline" className="gap-1.5">
                <CheckCircle2Icon className="h-3.5 w-3.5" />
                Type Safe
              </Badge>
            </div>
          </header>
        </FadeContent>

        {/* Tech Stack Section */}
        <FadeContent delay={200}>
          <section className="space-y-6">
            <div className="text-center">
              <h2 className="text-3xl font-bold tracking-tight">Tech Stack</h2>
              <p className="mt-2 text-muted-foreground">
                Built with modern tools and best practices
              </p>
            </div>

            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              {techStack.map((tech, index) => {
                const IconComponent = tech.icon;
                return (
                  <FadeContent key={tech.name} delay={300 + index * 50}>
                    <Card className="group flex flex-col h-full transition-all hover:shadow-lg hover:shadow-primary/5 hover:-translate-y-1">
                      <CardHeader>
                        <div className="flex items-center gap-3">
                          <IconComponent className="h-8 w-8 text-primary" />
                          <div>
                            <CardTitle className="text-lg">{tech.name}</CardTitle>
                          </div>
                        </div>
                      </CardHeader>
                      <CardContent className="flex-1">
                        <CardDescription>{tech.description}</CardDescription>
                      </CardContent>
                    </Card>
                  </FadeContent>
                );
              })}
            </div>
          </section>
        </FadeContent>

        {/* Features Section */}
        <FadeContent delay={400}>
          <section className="space-y-6">
            <div className="text-center">
              <h2 className="text-3xl font-bold tracking-tight">What's Included</h2>
              <p className="mt-2 text-muted-foreground">
                Everything you need to start building
              </p>
            </div>

            <div className="grid gap-6 md:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Palette className="h-6 w-6 text-primary" />
                    UI Components
                  </CardTitle>
                  <CardDescription>
                    25+ pre-built shadcn/ui components ready to use
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <ul className="space-y-2 text-sm">
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>Buttons, Cards, Forms, Dialogs</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>Tables, Tabs, Tooltips, Alerts</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>Toast notifications with Sonner</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>Animated components from React Bits</span>
                    </li>
                  </ul>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Zap className="h-6 w-6 text-primary" />
                    Developer Experience
                  </CardTitle>
                  <CardDescription>
                    Optimized workflow with modern tooling
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <ul className="space-y-2 text-sm">
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>Hot Module Replacement (HMR)</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>TypeScript strict mode enabled</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>ESLint + Prettier configured</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>Bundle analyzer included</span>
                    </li>
                  </ul>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Beaker className="h-6 w-6 text-primary" />
                    Testing Setup
                  </CardTitle>
                  <CardDescription>
                    Comprehensive testing with Vitest
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <ul className="space-y-2 text-sm">
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>Vitest + Testing Library</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>jsdom environment configured</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>Watch mode and UI mode</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>Example tests included</span>
                    </li>
                  </ul>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Rocket className="h-6 w-6 text-primary" />
                    Production Ready
                  </CardTitle>
                  <CardDescription>
                    Optimized for deployment
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <ul className="space-y-2 text-sm">
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>Tree shaking enabled</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>Code splitting configured</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>Minification and compression</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <CheckCircle2Icon className="h-4 w-4 text-green-500" />
                      <span>Environment variables support</span>
                    </li>
                  </ul>
                </CardContent>
              </Card>
            </div>
          </section>
        </FadeContent>

        {/* Commands Section */}
        <FadeContent delay={600}>
          <section className="space-y-6">
            <div className="text-center">
              <h2 className="text-3xl font-bold tracking-tight">Common Commands</h2>
              <p className="mt-2 text-muted-foreground">
                Makefile shortcuts for your workflow
              </p>
            </div>

            <Card>
              <CardContent className="pt-6">
                <div className="grid gap-3 sm:grid-cols-2">
                  {commands.map((item, index) => (
                    <FadeContent key={item.cmd} delay={700 + index * 30}>
                      <div className="group flex items-start gap-3 rounded-lg border border-border bg-muted/30 p-3 transition-all hover:bg-muted/50 hover:shadow-sm">
                        <code className="rounded bg-primary/10 px-2 py-1 text-xs font-mono text-primary">
                          {item.cmd}
                        </code>
                        <span className="text-sm text-muted-foreground">
                          {item.desc}
                        </span>
                      </div>
                    </FadeContent>
                  ))}
                </div>
              </CardContent>
            </Card>
          </section>
        </FadeContent>

        {/* CTA Section */}
        <FadeContent delay={800}>
          <section className="text-center">
            <Card className="border-primary/20 bg-gradient-to-br from-primary/5 to-primary/10">
              <CardContent className="pt-6">
                <div className="flex flex-col items-center gap-4">
                  <h3 className="text-2xl font-bold">Ready to Build?</h3>
                  <p className="text-muted-foreground max-w-2xl">
                    Start developing your next project with this modern React template.
                    Visit the <span className="font-semibold text-foreground">About</span> page to see all UI components in action.
                  </p>
                  <div className="flex gap-2">
                    <Badge variant="secondary" className="text-xs">
                      MIT Licensed
                    </Badge>
                    <Badge variant="secondary" className="text-xs">
                      Open Source
                    </Badge>
                  </div>
                </div>
              </CardContent>
            </Card>
          </section>
        </FadeContent>
      </div>
    );
}

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { Alert, AlertDescription, AlertTitle } from "../components/ui/alert";
import { Separator } from "../components/ui/separator";
import FadeContent from "../components/FadeContent";
import { 
  Package, 
  Download, 
  Code, 
  BookOpen,
  Terminal,
  Sparkles,
  Palette,
  Bell,
  Database,
  Search,
  ExternalLink,
  Check,
  X,
  Home,
  Settings,
  User,
  Heart,
  Star,
  Trash,
  Edit
} from "lucide-react";

export function AddonsPage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-16">
        <FadeContent>
          <header className="flex flex-col gap-3">
            <div className="flex items-center gap-3">
              <Package className="h-8 w-8 text-primary" />
              <h1 className="text-4xl font-bold tracking-tight">Addons & Libraries</h1>
            </div>
            <p className="text-lg text-muted-foreground">
              Learn how to use the integrated libraries and install new components
            </p>
          </header>
        </FadeContent>

        <Tabs defaultValue="shadcn" className="w-full">
          <TabsList className="grid w-full grid-cols-5">
            <TabsTrigger value="shadcn">shadcn/ui</TabsTrigger>
            <TabsTrigger value="reactbits">React Bits</TabsTrigger>
            <TabsTrigger value="zustand">Zustand</TabsTrigger>
            <TabsTrigger value="sonner">Sonner</TabsTrigger>
            <TabsTrigger value="icons">Icons</TabsTrigger>
          </TabsList>

          {/* shadcn/ui Tab */}
          <TabsContent value="shadcn" className="space-y-6">
            <FadeContent delay={100}>
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-2">
                    <Palette className="h-5 w-5 text-primary" />
                    <CardTitle>shadcn/ui Components</CardTitle>
                  </div>
                  <CardDescription>
                    Beautifully designed components built with Radix UI and Tailwind CSS
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <Package className="h-5 w-5" />
                      Installed Components
                    </h3>
                    <div className="flex flex-wrap gap-2">
                      {[
                        "button", "card", "input", "label", "select", "textarea",
                        "dialog", "dropdown-menu", "form", "checkbox", "radio-group",
                        "switch", "tabs", "sonner", "avatar", "badge", "separator",
                        "skeleton", "alert", "alert-dialog", "table", "popover",
                        "tooltip", "sheet", "accordion"
                      ].map(comp => (
                        <Badge key={comp} variant="secondary">{comp}</Badge>
                      ))}
                    </div>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <Download className="h-5 w-5" />
                      Installing New Components
                    </h3>
                    <Alert>
                      <Terminal className="h-4 w-4" />
                      <AlertTitle>CLI Command</AlertTitle>
                      <AlertDescription>
                        <code className="text-sm">npx shadcn@latest add [component-name]</code>
                      </AlertDescription>
                    </Alert>
                    <div className="space-y-2">
                      <p className="text-sm text-muted-foreground">Examples:</p>
                      <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`# Install a single component
npx shadcn@latest add button

# Install multiple components
npx shadcn@latest add button card input

# Browse all available components
npx shadcn@latest add`}
                      </pre>
                    </div>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <Code className="h-5 w-5" />
                      Usage Example
                    </h3>
                    <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`import { Button } from "../components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card"

export function MyComponent() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Hello World</CardTitle>
      </CardHeader>
      <CardContent>
        <Button>Click me</Button>
      </CardContent>
    </Card>
  )
}`}
                    </pre>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <BookOpen className="h-5 w-5" />
                      Resources
                    </h3>
                    <ul className="space-y-2 text-sm">
                      <li>
                        <a 
                          href="https://ui.shadcn.com" 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-primary hover:underline flex items-center gap-1"
                        >
                          <BookOpen className="h-4 w-4" />
                          Official Documentation
                        </a>
                      </li>
                      <li>
                        <a 
                          href="https://ui.shadcn.com/docs/components" 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-primary hover:underline flex items-center gap-1"
                        >
                          <Palette className="h-4 w-4" />
                          Component Gallery
                        </a>
                      </li>
                      <li>
                        <a 
                          href="https://ui.shadcn.com/themes" 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-primary hover:underline flex items-center gap-1"
                        >
                          <Sparkles className="h-4 w-4" />
                          Themes & Customization
                        </a>
                      </li>
                    </ul>
                  </div>
                </CardContent>
              </Card>
            </FadeContent>
          </TabsContent>

          {/* React Bits Tab */}
          <TabsContent value="reactbits" className="space-y-6">
            <FadeContent delay={100}>
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-2">
                    <Sparkles className="h-5 w-5 text-primary" />
                    <CardTitle>React Bits Animated Components</CardTitle>
                  </div>
                  <CardDescription>
                    Beautiful animated components for modern React applications
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <Sparkles className="h-5 w-5" />
                      Installed Components
                    </h3>
                    <div className="flex flex-wrap gap-2">
                      {[
                        "BlurText", "GradientText", "FadeContent", 
                        "ShinyText", "SplitText", "AnimatedList"
                      ].map(comp => (
                        <Badge key={comp} variant="secondary">{comp}</Badge>
                      ))}
                    </div>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <Download className="h-5 w-5" />
                      Installing New Components
                    </h3>
                    <Alert>
                      <Terminal className="h-4 w-4" />
                      <AlertTitle>CLI Command</AlertTitle>
                      <AlertDescription>
                        <code className="text-sm">npx shadcn@latest add https://reactbits.dev/r/[Component]-TS-TW</code>
                      </AlertDescription>
                    </Alert>
                    <div className="space-y-2">
                      <p className="text-sm text-muted-foreground">Popular components:</p>
                      <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`# Text animations
npx shadcn@latest add https://reactbits.dev/r/BlurText-TS-TW
npx shadcn@latest add https://reactbits.dev/r/GradientText-TS-TW
npx shadcn@latest add https://reactbits.dev/r/ShinyText-TS-TW
npx shadcn@latest add https://reactbits.dev/r/RotatingText-TS-TW

# Content animations
npx shadcn@latest add https://reactbits.dev/r/FadeContent-TS-TW
npx shadcn@latest add https://reactbits.dev/r/AnimatedList-TS-TW

# Effects
npx shadcn@latest add https://reactbits.dev/r/MagnetLines-TS-TW
npx shadcn@latest add https://reactbits.dev/r/ParticleEffect-TS-TW`}
                      </pre>
                    </div>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <Code className="h-5 w-5" />
                      Usage Examples
                    </h3>
                    <div className="space-y-4">
                      <div>
                        <p className="text-sm font-medium mb-2">BlurText</p>
                        <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`import BlurText from "../components/BlurText"

<BlurText 
  text="Hello World" 
  className="text-4xl font-bold"
  delay={50}
/>`}
                        </pre>
                      </div>
                      <div>
                        <p className="text-sm font-medium mb-2">GradientText</p>
                        <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`import GradientText from "../components/GradientText"

<GradientText 
  colors={["#a855f7", "#ec4899", "#f97316"]}
  className="text-2xl"
>
  Animated Gradient
</GradientText>`}
                        </pre>
                      </div>
                      <div>
                        <p className="text-sm font-medium mb-2">FadeContent</p>
                        <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`import FadeContent from "../components/FadeContent"

<FadeContent delay={200}>
  <div>Content that fades in</div>
</FadeContent>`}
                        </pre>
                      </div>
                    </div>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <BookOpen className="h-5 w-5" />
                      Resources
                    </h3>
                    <ul className="space-y-2 text-sm">
                      <li>
                        <a 
                          href="https://reactbits.dev" 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-primary hover:underline flex items-center gap-1"
                        >
                          <ExternalLink className="h-4 w-4" />
                          React Bits Website
                        </a>
                      </li>
                      <li>
                        <a 
                          href="https://reactbits.dev/components" 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-primary hover:underline flex items-center gap-1"
                        >
                          <Sparkles className="h-4 w-4" />
                          Component Gallery
                        </a>
                      </li>
                    </ul>
                  </div>
                </CardContent>
              </Card>
            </FadeContent>
          </TabsContent>

          {/* Zustand Tab */}
          <TabsContent value="zustand" className="space-y-6">
            <FadeContent delay={100}>
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-2">
                    <Database className="h-5 w-5 text-primary" />
                    <CardTitle>Zustand State Management</CardTitle>
                  </div>
                  <CardDescription>
                    A small, fast and scalable bearbones state-management solution
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <Database className="h-5 w-5" />
                      What is Zustand?
                    </h3>
                    <p className="text-sm text-muted-foreground">
                      Zustand is a lightweight state management library that provides a simple API 
                      for managing global state in React applications. No providers, no boilerplate.
                    </p>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <Code className="h-5 w-5" />
                      Creating a Store
                    </h3>
                    <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`// src/stores/useCounterStore.ts
import { create } from 'zustand'

interface CounterState {
  count: number
  increment: () => void
  decrement: () => void
  reset: () => void
}

export const useCounterStore = create<CounterState>((set) => ({
  count: 0,
  increment: () => set((state) => ({ count: state.count + 1 })),
  decrement: () => set((state) => ({ count: state.count - 1 })),
  reset: () => set({ count: 0 }),
}))`}
                    </pre>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold">Using the Store</h3>
                    <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`// In your component
import { useCounterStore } from '../stores/useCounterStore'

export function Counter() {
  const { count, increment, decrement, reset } = useCounterStore()
  
  return (
    <div>
      <p>Count: {count}</p>
      <button onClick={increment}>+</button>
      <button onClick={decrement}>-</button>
      <button onClick={reset}>Reset</button>
    </div>
  )
}

// Select only what you need (prevents unnecessary re-renders)
export function CountDisplay() {
  const count = useCounterStore((state) => state.count)
  return <p>{count}</p>
}`}
                    </pre>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold">Advanced Features</h3>
                    <div className="space-y-4">
                      <div>
                        <p className="text-sm font-medium mb-2 flex items-center gap-2">
                          <Database className="h-4 w-4" />
                          Persist State
                        </p>
                        <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export const useStore = create(
  persist(
    (set) => ({
      user: null,
      setUser: (user) => set({ user }),
    }),
    { name: 'user-storage' }
  )
)`}
                        </pre>
                      </div>
                      <div>
                        <p className="text-sm font-medium mb-2 flex items-center gap-2">
                          <Code className="h-4 w-4" />
                          Immer Middleware
                        </p>
                        <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`import { create } from 'zustand'
import { immer } from 'zustand/middleware/immer'

export const useStore = create(
  immer((set) => ({
    todos: [],
    addTodo: (todo) => set((state) => {
      state.todos.push(todo)
    }),
  }))
)`}
                        </pre>
                      </div>
                    </div>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <BookOpen className="h-5 w-5" />
                      Resources
                    </h3>
                    <ul className="space-y-2 text-sm">
                      <li>
                        <a 
                          href="https://zustand-demo.pmnd.rs" 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-primary hover:underline flex items-center gap-1"
                        >
                          <BookOpen className="h-4 w-4" />
                          Official Documentation
                        </a>
                      </li>
                      <li>
                        <a 
                          href="https://github.com/pmndrs/zustand" 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-primary hover:underline flex items-center gap-1"
                        >
                          <Code className="h-4 w-4" />
                          GitHub Repository
                        </a>
                      </li>
                    </ul>
                  </div>
                </CardContent>
              </Card>
            </FadeContent>
          </TabsContent>

          {/* Sonner Tab */}
          <TabsContent value="sonner" className="space-y-6">
            <FadeContent delay={100}>
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-2">
                    <Bell className="h-5 w-5 text-primary" />
                    <CardTitle>Sonner Toast Notifications</CardTitle>
                  </div>
                  <CardDescription>
                    An opinionated toast component for React
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <Bell className="h-5 w-5" />
                      What is Sonner?
                    </h3>
                    <p className="text-sm text-muted-foreground">
                      Sonner is a beautiful, accessible toast notification library with a simple API. 
                      It's already integrated with shadcn/ui theming.
                    </p>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <Code className="h-5 w-5" />
                      Setup (Already Done)
                    </h3>
                    <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`// In App.tsx (already added)
import { Toaster } from "../components/ui/sonner"

function App() {
  return (
    <>
      {/* Your app content */}
      <Toaster />
    </>
  )
}`}
                    </pre>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold">Usage Examples</h3>
                    <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`import { toast } from "sonner"

// Basic toast
toast("Event has been created")

// With description
toast("Event has been created", {
  description: "Monday, January 3rd at 6:00pm",
})

// Success toast
toast.success("Profile updated", {
  description: "Your changes have been saved",
})

// Error toast
toast.error("Something went wrong", {
  description: "Please try again later",
})

// Info toast
toast.info("New update available", {
  description: "Version 2.0 is ready to install",
})

// Warning toast
toast.warning("Storage almost full", {
  description: "You have used 90% of your storage",
})

// Loading toast
const toastId = toast.loading("Uploading...")
// Later...
toast.success("Upload complete", { id: toastId })

// With action button
toast("Event created", {
  action: {
    label: "Undo",
    onClick: () => console.log("Undo"),
  },
})

// Promise toast
toast.promise(
  fetch("/api/data"),
  {
    loading: "Loading...",
    success: "Data loaded",
    error: "Failed to load",
  }
)`}
                    </pre>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold">Customization</h3>
                    <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`// Custom duration
toast("Quick message", { duration: 2000 })

// Custom position
<Toaster position="top-right" />

// Rich content
toast(
  <div>
    <h3>Custom Toast</h3>
    <p>With custom JSX content</p>
  </div>
)

// Dismiss programmatically
const id = toast("Message")
toast.dismiss(id)

// Dismiss all
toast.dismiss()`}
                    </pre>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <BookOpen className="h-5 w-5" />
                      Resources
                    </h3>
                    <ul className="space-y-2 text-sm">
                      <li>
                        <a 
                          href="https://sonner.emilkowal.ski" 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-primary hover:underline flex items-center gap-1"
                        >
                          <BookOpen className="h-4 w-4" />
                          Official Documentation
                        </a>
                      </li>
                      <li>
                        <a 
                          href="https://github.com/emilkowalski/sonner" 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-primary hover:underline flex items-center gap-1"
                        >
                          <Code className="h-4 w-4" />
                          GitHub Repository
                        </a>
                      </li>
                    </ul>
                  </div>
                </CardContent>
              </Card>
            </FadeContent>
          </TabsContent>

          {/* Icons Tab */}
          <TabsContent value="icons" className="space-y-6">
            <FadeContent delay={100}>
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-2">
                    <Sparkles className="h-5 w-5 text-primary" />
                    <CardTitle>Lucide React Icons</CardTitle>
                  </div>
                  <CardDescription>
                    Beautiful & consistent icon toolkit with 1000+ icons
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <Sparkles className="h-5 w-5" />
                      What is Lucide?
                    </h3>
                    <p className="text-sm text-muted-foreground">
                      Lucide is a community-run fork of Feather Icons, providing a comprehensive 
                      set of beautiful, consistent icons for React applications.
                    </p>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <Code className="h-5 w-5" />
                      Usage
                    </h3>
                    <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`import { 
  CheckIcon, 
  XIcon, 
  HomeIcon,
  SettingsIcon,
  UserIcon 
} from "lucide-react"

export function MyComponent() {
  return (
    <div>
      <CheckIcon className="h-4 w-4" />
      <XIcon className="h-6 w-6 text-red-500" />
      <HomeIcon className="h-8 w-8 text-blue-500" />
    </div>
  )
}`}
                    </pre>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold">Common Icons</h3>
                    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
                      {[
                        { name: "Check", icon: Check },
                        { name: "X", icon: X },
                        { name: "Home", icon: Home },
                        { name: "Settings", icon: Settings },
                        { name: "User", icon: User },
                        { name: "Search", icon: Search },
                        { name: "Bell", icon: Bell },
                        { name: "Heart", icon: Heart },
                        { name: "Star", icon: Star },
                        { name: "Trash", icon: Trash },
                        { name: "Edit", icon: Edit },
                        { name: "Download", icon: Download },
                      ].map(({ name, icon: IconComponent }) => (
                        <div key={name} className="flex items-center gap-2 text-sm">
                          <IconComponent className="h-5 w-5 text-primary" />
                          <code className="text-xs">{name}</code>
                        </div>
                      ))}
                    </div>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold">Customization</h3>
                    <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
{`// Size
<CheckIcon className="h-4 w-4" />  // 16px
<CheckIcon className="h-6 w-6" />  // 24px
<CheckIcon className="h-8 w-8" />  // 32px

// Color
<CheckIcon className="text-red-500" />
<CheckIcon className="text-primary" />

// Stroke width
<CheckIcon strokeWidth={1.5} />
<CheckIcon strokeWidth={2.5} />

// Animation
<CheckIcon className="animate-spin" />
<CheckIcon className="animate-pulse" />`}
                    </pre>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <BookOpen className="h-5 w-5" />
                      Resources
                    </h3>
                    <ul className="space-y-2 text-sm">
                      <li>
                        <a 
                          href="https://lucide.dev" 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-primary hover:underline flex items-center gap-1"
                        >
                          <Search className="h-4 w-4" />
                          Icon Search & Documentation
                        </a>
                      </li>
                      <li>
                        <a 
                          href="https://lucide.dev/guide/packages/lucide-react" 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-primary hover:underline flex items-center gap-1"
                        >
                          <BookOpen className="h-4 w-4" />
                          React Guide
                        </a>
                      </li>
                      <li>
                        <a 
                          href="https://github.com/lucide-icons/lucide" 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-primary hover:underline flex items-center gap-1"
                        >
                          <Code className="h-4 w-4" />
                          GitHub Repository
                        </a>
                      </li>
                    </ul>
                  </div>
                </CardContent>
              </Card>
            </FadeContent>
          </TabsContent>
        </Tabs>

        {/* Quick Reference Card */}
        <FadeContent delay={200}>
          <Card className="border-primary/20 bg-gradient-to-br from-primary/5 to-primary/10">
            <CardHeader>
              <CardTitle>Quick Reference</CardTitle>
              <CardDescription>Common commands for adding components</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="space-y-2">
                  <p className="text-sm font-medium">shadcn/ui</p>
                  <code className="block rounded bg-background/50 px-3 py-2 text-xs">
                    npx shadcn@latest add [component]
                  </code>
                </div>
                <div className="space-y-2">
                  <p className="text-sm font-medium">React Bits</p>
                  <code className="block rounded bg-background/50 px-3 py-2 text-xs">
                    npx shadcn@latest add https://reactbits.dev/r/[Component]-TS-TW
                  </code>
                </div>
              </div>
            </CardContent>
          </Card>
        </FadeContent>
      </div>
    );
}

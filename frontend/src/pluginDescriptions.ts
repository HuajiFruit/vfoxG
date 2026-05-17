import { currentLang } from './i18n';

const descriptions: Record<string, { en: string, zh: string }> = {
  'clang': {
    en: 'A C language family frontend for LLVM.',
    zh: '基于 LLVM 的 C 语言家族编译器前端。'
  },
  'cmake': {
    en: 'An open-source, cross-platform family of tools designed to build, test and package software.',
    zh: '一个开源的、跨平台的自动化构建系统，用来管理软件构建的过程。'
  },
  'dart': {
    en: 'A client-optimized language for fast apps on any platform.',
    zh: '由 Google 开发的客户端优化语言，用于在任何平台上快速构建应用。'
  },
  'deno': {
    en: 'A modern runtime for JavaScript and TypeScript.',
    zh: '一个现代化的 JavaScript 和 TypeScript 运行时。'
  },
  'dotnet': {
    en: 'A free, cross-platform, open source developer platform for building many different types of applications.',
    zh: '一个免费、跨平台、开源的开发者平台，用于构建多种不同类型的应用。'
  },
  'elixir': {
    en: 'A dynamic, functional language designed for building scalable and maintainable applications.',
    zh: '一种动态的函数式编程语言，设计用于构建可扩展和易维护的应用。'
  },
  'erlang': {
    en: 'A programming language used to build massively scalable soft real-time systems.',
    zh: '一种用于构建大规模可扩展软实时系统的编程语言。'
  },
  'etcd': {
    en: 'A distributed, reliable key-value store for the most critical data of a distributed system.',
    zh: '一个分布式的、可靠的键值存储系统，用于存储分布式系统中最关键的数据。'
  },
  'flutter': {
    en: 'Google\'s UI toolkit for building beautiful, natively compiled applications for mobile, web, and desktop from a single codebase.',
    zh: 'Google 的 UI 工具包，用于通过单一代码库构建精美的、原生编译的跨平台应用。'
  },
  'golang': {
    en: 'An open source programming language supported by Google.',
    zh: '由 Google 支持的开源编程语言，以其并发机制和高效性能著称。'
  },
  'gradle': {
    en: 'A powerful build system for the JVM.',
    zh: '一个基于 JVM 的强大的自动化构建工具。'
  },
  'groovy': {
    en: 'A multi-faceted language for the Java platform.',
    zh: '一种基于 Java 平台的灵活、动态的多面手编程语言。'
  },
  'java': {
    en: 'A high-level, class-based, object-oriented programming language.',
    zh: '一种高级、基于类的面向对象编程语言。'
  },
  'kotlin': {
    en: 'A modern programming language that makes developers happier.',
    zh: '一种让开发者更快乐的现代编程语言，完全兼容 Java。'
  },
  'maven': {
    en: 'A software project management and comprehension tool.',
    zh: '一个软件项目管理和理解工具，基于项目对象模型 (POM) 的概念。'
  },
  'nodejs': {
    en: 'An asynchronous event-driven JavaScript runtime.',
    zh: '一个基于 Chrome V8 引擎的异步事件驱动 JavaScript 运行时。'
  },
  'php': {
    en: 'A popular general-purpose scripting language that is especially suited to web development.',
    zh: '一种流行的通用脚本语言，特别适合于 Web 开发。'
  },
  'python': {
    en: 'A programming language that lets you work quickly and integrate systems more effectively.',
    zh: '一种让你能够快速工作并更有效地整合系统的编程语言。'
  },
  'scala': {
    en: 'A programming language that combines object-oriented and functional programming in one concise, high-level language.',
    zh: '一种将面向对象和函数式编程结合在一种简洁的高级语言中的编程语言。'
  },
  'zig': {
    en: 'A general-purpose programming language and toolchain for maintaining robust, optimal, and reusable software.',
    zh: '一种通用的编程语言和工具链，用于维护健壮、最优且可重用的软件。'
  },
  'bun': {
    en: 'A fast all-in-one JavaScript runtime.',
    zh: '一个快速的 JavaScript 全能运行时、打包器、转译器和包管理器。'
  },
  'kubectl': {
    en: 'The Kubernetes command-line tool.',
    zh: 'Kubernetes 命令行工具，让您可以对 Kubernetes 集群运行命令。'
  },
  'mongo': {
    en: 'MongoDB shell environment.',
    zh: 'MongoDB 数据库的交互式 JavaScript shell。'
  },
  'mongod': {
    en: 'MongoDB database daemon.',
    zh: 'MongoDB 数据库的核心后台守护进程。'
  },
  'ruby': {
    en: 'A dynamic, open source programming language with a focus on simplicity and productivity.',
    zh: '一种动态、开源的编程语言，注重简单性和生产力。'
  },
  'rust': {
    en: 'A language empowering everyone to build reliable and efficient software.',
    zh: '一门赋予每个人构建可靠且高效软件能力的编程语言。'
  },
  'terraform': {
    en: 'An infrastructure as code tool that lets you build, change, and version cloud and on-prem resources safely and efficiently.',
    zh: '一种基础设施即代码工具，可让您安全高效地构建、更改和管理云资源版本。'
  },
  'tomcat': {
    en: 'An open source implementation of the Java EE platform.',
    zh: 'Java EE 平台的开源实现，一个流行的 Web 应用服务器。'
  }
};

export const getPluginDescription = (name: string): string => {
  const desc = descriptions[name.toLowerCase()];
  if (!desc) return '';
  return desc[currentLang.value] || desc.en;
};

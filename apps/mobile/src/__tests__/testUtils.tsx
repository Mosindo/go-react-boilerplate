import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react-native";
import { ThemeProvider } from "../theme/ThemeProvider";
import { ToastProvider } from "../ui";

export async function renderWithProviders(ui: React.ReactElement) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false } }
  });
  const result = await render(
    <ThemeProvider>
      <QueryClientProvider client={client}>
        <ToastProvider>{ui}</ToastProvider>
      </QueryClientProvider>
    </ThemeProvider>
  );
  return { client, ...result };
}

export function navStub() {
  return {
    navigate: jest.fn(),
    goBack: jest.fn(),
    replace: jest.fn(),
    popToTop: jest.fn(),
    setOptions: jest.fn(),
    addListener: jest.fn(() => jest.fn())
  };
}

/** Builds screen props from a navigation stub and route without fighting React Navigation's generics. */
export function screenProps<P>(route: { name: string; params?: object }, navigation = navStub()): P {
  return { navigation, route: { key: `${route.name}-key`, ...route } } as unknown as P;
}

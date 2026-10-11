/** Pure-logic tests only (no native runtime): plain jest + ts-jest. */
module.exports = {
  testEnvironment: 'node',
  testMatch: ['<rootDir>/src/**/*.test.ts'],
  transform: {
    '^.+\\.ts$': ['ts-jest', { tsconfig: 'tsconfig.jest.json', diagnostics: true }],
  },
};

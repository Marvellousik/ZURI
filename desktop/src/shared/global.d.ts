import { IZuriAPI } from './types';

declare global {
  interface Window {
    zuriAPI: IZuriAPI;
  }
}

export {};

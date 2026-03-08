#!/usr/bin/env node
// Compatibility shim — openclaw.mjs redirects to blackcat.mjs
// This file exists because many internal modules still reference it.
// TODO: Remove after full source rebrand (separate effort).
import "./blackcat.mjs";
